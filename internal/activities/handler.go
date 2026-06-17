package activities

import (
	"encoding/json"
	"issuetrack/internal/users"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type Handlers struct {
	store *Store
}

func NewHandlers(store *Store) *Handlers {
	return &Handlers{store: store}
}
func (h *Handlers) ListByProject(w http.ResponseWriter, r *http.Request) {
	pidStr := chi.URLParam(r, "projectId")
	projectID, _ := strconv.Atoi(pidStr)
	uid, _ := users.UserIDFromContext(r.Context())

	list, err := h.store.ListByProject(r.Context(), projectID, uid, 50)
	if err != nil {
		http.Error(w, "list failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}
