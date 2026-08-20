package httpapi

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	coreauth "github.com/switchonyourcode/switchonyourcode/backend/internal/auth"
)

const (
	sessionCookieName = "switchonyourcode_session"
	csrfCookieName    = "switchonyourcode_csrf"
	maxAuthBodyBytes  = 32 << 10
)

type AuthOptions struct {
	SecureCookies bool
}

type authHandlers struct {
	logger        *slog.Logger
	service       *coreauth.Service
	secureCookies bool
}

type authenticatedRequest struct {
	Session coreauth.Session
	Token   string
}

type authenticatedRequestKey struct{}

type bootstrapRequest struct {
	Email            string `json:"email"`
	DisplayName      string `json:"display_name"`
	Password         string `json:"password"`
	OrganisationName string `json:"organisation_name"`
	OrganisationSlug string `json:"organisation_slug"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userResponse struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}

type organisationResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
	Role string `json:"role"`
}

type principalResponse struct {
	User          userResponse           `json:"user"`
	Organisations []organisationResponse `json:"organisations"`
}

type apiErrorResponse struct {
	Error apiError `json:"error"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func newAuthHandlers(logger *slog.Logger, service *coreauth.Service, options AuthOptions) *authHandlers {
	return &authHandlers{logger: logger, service: service, secureCookies: options.SecureCookies}
}

func (h *authHandlers) bootstrapStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	required, err := h.service.BootstrapRequired(r.Context())
	if err != nil {
		h.internalError(w, r, "check bootstrap status", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"required": required})
}

func (h *authHandlers) bootstrap(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	var request bootstrapRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "Request body must be valid JSON.")
		return
	}

	issued, err := h.service.Bootstrap(r.Context(), coreauth.BootstrapInput{
		Email:            request.Email,
		DisplayName:      request.DisplayName,
		Password:         request.Password,
		OrganisationName: request.OrganisationName,
		OrganisationSlug: request.OrganisationSlug,
	})
	if err != nil {
		var validationErr *coreauth.ValidationError
		switch {
		case errors.As(err, &validationErr):
			writeAPIError(w, http.StatusBadRequest, "validation_error", validationErr.Error())
		case errors.Is(err, coreauth.ErrBootstrapComplete):
			writeAPIError(w, http.StatusConflict, "bootstrap_complete", "Initial setup has already been completed.")
		default:
			h.internalError(w, r, "bootstrap instance", err)
		}
		return
	}

	h.setSessionCookies(w, issued)
	writeJSON(w, http.StatusCreated, principalFromCore(issued.Principal))
}

func (h *authHandlers) login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	var request loginRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "Request body must be valid JSON.")
		return
	}

	issued, err := h.service.Login(r.Context(), coreauth.LoginInput{Email: request.Email, Password: request.Password})
	if err != nil {
		if errors.Is(err, coreauth.ErrInvalidCredentials) {
			writeAPIError(w, http.StatusUnauthorized, "invalid_credentials", "Email or password is incorrect.")
			return
		}
		h.internalError(w, r, "authenticate user", err)
		return
	}

	h.setSessionCookies(w, issued)
	writeJSON(w, http.StatusOK, principalFromCore(issued.Principal))
}

func (h *authHandlers) me(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	authenticated, ok := authenticatedFromContext(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "unauthenticated", "Authentication is required.")
		return
	}
	writeJSON(w, http.StatusOK, principalFromCore(authenticated.Session.Principal))
}

func (h *authHandlers) logout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	authenticated, ok := authenticatedFromContext(r.Context())
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "unauthenticated", "Authentication is required.")
		return
	}
	if err := h.service.Logout(r.Context(), authenticated.Token); err != nil {
		h.internalError(w, r, "delete session", err)
		return
	}
	h.clearSessionCookies(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *authHandlers) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil || cookie.Value == "" {
			writeAPIError(w, http.StatusUnauthorized, "unauthenticated", "Authentication is required.")
			return
		}

		session, err := h.service.AuthenticateSession(r.Context(), cookie.Value)
		if err != nil {
			if errors.Is(err, coreauth.ErrSessionNotFound) {
				h.clearSessionCookies(w)
				writeAPIError(w, http.StatusUnauthorized, "unauthenticated", "Authentication is required.")
				return
			}
			h.internalError(w, r, "authenticate session", err)
			return
		}

		ctx := context.WithValue(r.Context(), authenticatedRequestKey{}, authenticatedRequest{Session: session, Token: cookie.Value})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *authHandlers) requireCSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authenticated, ok := authenticatedFromContext(r.Context())
		if !ok {
			writeAPIError(w, http.StatusUnauthorized, "unauthenticated", "Authentication is required.")
			return
		}

		cookie, err := r.Cookie(csrfCookieName)
		header := r.Header.Get("X-CSRF-Token")
		if err != nil || header == "" || !constantTimeStringEqual(cookie.Value, header) || !coreauth.ValidCSRF(authenticated.Session, header) {
			writeAPIError(w, http.StatusForbidden, "csrf_failed", "CSRF validation failed.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *authHandlers) setSessionCookies(w http.ResponseWriter, issued coreauth.IssuedSession) {
	maxAge := int(time.Until(issued.ExpiresAt).Seconds())
	if maxAge < 1 {
		maxAge = 1
	}
	h.setCookie(w, sessionCookieName, issued.SessionToken, issued.ExpiresAt, maxAge, true)
	h.setCookie(w, csrfCookieName, issued.CSRFToken, issued.ExpiresAt, maxAge, false)
}

func (h *authHandlers) clearSessionCookies(w http.ResponseWriter) {
	expired := time.Unix(1, 0).UTC()
	h.setCookie(w, sessionCookieName, "", expired, -1, true)
	h.setCookie(w, csrfCookieName, "", expired, -1, false)
}

func (h *authHandlers) setCookie(w http.ResponseWriter, name, value string, expires time.Time, maxAge int, httpOnly bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		Expires:  expires,
		MaxAge:   maxAge,
		HttpOnly: httpOnly,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *authHandlers) internalError(w http.ResponseWriter, r *http.Request, operation string, err error) {
	h.logger.ErrorContext(r.Context(), "authentication request failed", "operation", operation, "error", err)
	writeAPIError(w, http.StatusInternalServerError, "internal_error", "The request could not be completed.")
}

func authenticatedFromContext(ctx context.Context) (authenticatedRequest, bool) {
	value, ok := ctx.Value(authenticatedRequestKey{}).(authenticatedRequest)
	return value, ok
}

func principalFromCore(principal coreauth.Principal) principalResponse {
	organisations := make([]organisationResponse, 0, len(principal.Organisations))
	for _, organisation := range principal.Organisations {
		organisations = append(organisations, organisationResponse{
			ID: organisation.ID, Name: organisation.Name, Slug: organisation.Slug, Role: organisation.Role,
		})
	}
	return principalResponse{
		User:          userResponse{ID: principal.User.ID, Email: principal.User.Email, DisplayName: principal.User.DisplayName},
		Organisations: organisations,
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxAuthBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain a single JSON object")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeAPIError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, apiErrorResponse{Error: apiError{Code: code, Message: message}})
}

func constantTimeStringEqual(left, right string) bool {
	if len(left) != len(right) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}
