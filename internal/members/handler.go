package members

import (
	"encoding/json"
	"issuetrack/internal/users"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	store *Store
}

func NewHandlers(store *Store) *Handler {
	return &Handler{store: store}
}
func (h *Handler) Invite(w http.ResponseWriter, r *http.Request) {
	pidStr := chi.URLParam(r, "projectId")
	projectID, _ := strconv.Atoi(pidStr)

	var req struct {
		Username string `json:"username"`
		Role     string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	uid, _ := users.UserIDFromContext(r.Context())
	invite, err := h.store.Invite(r.Context(), projectID, uid, req.Username, req.Role)
	if err != nil {
		http.Error(w, "invite failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(invite)
}
func (h *Handler) ListByProject(w http.ResponseWriter, r *http.Request) {
	pidStr := chi.URLParam(r, "projectId")
	projectID, _ := strconv.Atoi(pidStr)

	uid, _ := users.UserIDFromContext(r.Context())
	list, err := h.store.ListByProject(r.Context(), projectID, uid)
	if err != nil {
		http.Error(w, "list failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}
func (h *Handler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	pidStr := chi.URLParam(r, "projectId")
	projectID, _ := strconv.Atoi(pidStr)
	uidStr := chi.URLParam(r, "userId")
	userID, _ := strconv.Atoi(uidStr)

	var req struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	uid, _ := users.UserIDFromContext(r.Context())
	if err := h.store.UpdateRole(r.Context(), projectID, userID, uid, req.Role); err != nil {
		http.Error(w, "update failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *Handler) Remove(w http.ResponseWriter, r *http.Request) {
	pidStr := chi.URLParam(r, "projectId")
	projectID, _ := strconv.Atoi(pidStr)
	uidStr := chi.URLParam(r, "userId")
	userID, _ := strconv.Atoi(uidStr)

	uid, _ := users.UserIDFromContext(r.Context())
	if err := h.store.Remove(r.Context(), projectID, userID, uid); err != nil {
		http.Error(w, "remove failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
