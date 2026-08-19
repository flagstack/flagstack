package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

const readinessTimeout = 2 * time.Second

func readinessHandler(checker readinessChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := http.StatusOK
		response := healthResponse{Status: "ok"}

		if checker == nil {
			status = http.StatusServiceUnavailable
			response.Status = "unavailable"
		} else {
			ctx, cancel := context.WithTimeout(r.Context(), readinessTimeout)
			defer cancel()
			if err := checker.Ping(ctx); err != nil {
				status = http.StatusServiceUnavailable
				response.Status = "unavailable"
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(response)
	}
}
