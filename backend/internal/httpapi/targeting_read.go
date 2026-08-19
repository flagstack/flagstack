package httpapi

import "net/http"

func (h *targetingHandlers) getFlagTargeting(w http.ResponseWriter, r *http.Request) {
	membership, ok := targetingMembership(w, r, false)
	if !ok {
		return
	}
	state, err := h.service.GetFlagState(r.Context(), membership.ID, r.PathValue("project"), r.PathValue("featureFlag"))
	if err != nil {
		writeTargetingError(w, err, "Feature flag targeting could not be loaded.")
		return
	}
	writeJSON(w, http.StatusOK, flagTargetingFromCore(state))
}
