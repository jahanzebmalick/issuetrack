package tags

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
func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	pidStr := chi.URLParam(r, "projectId")
	projectID, err := strconv.Atoi(pidStr)
	if err != nil {
		http.Error(w, "invalid project id", http.StatusBadRequest)
		return
	}
	var req struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	uid, _ := users.UserIDFromContext(r.Context())
	tag, err := h.store.Create(r.Context(), projectID, uid, req.Name, req.Color)
	if err != nil {
		http.Error(w, "create failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tag)
}
func (h *Handlers) ListByProject(w http.ResponseWriter, r *http.Request) {
	pidStr := chi.URLParam(r, "projectId")
	projectID, err := strconv.Atoi(pidStr)
	if err != nil {
		http.Error(w, "invalid project id", http.StatusBadRequest)
		return
	}
	uid, _ := users.UserIDFromContext(r.Context())
	list, err := h.store.ListByProject(r.Context(), projectID, uid)
	if err != nil {
		http.Error(w, "list failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}
func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	tagID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid tag id", http.StatusBadRequest)
		return
	}
	uid, _ := users.UserIDFromContext(r.Context())
	if err := h.store.Delete(r.Context(), tagID, uid); err != nil {
		http.Error(w, "delete failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *Handlers) Attach(w http.ResponseWriter, r *http.Request) {
	issueidStr := chi.URLParam(r, "issueId")
	issueID, err := strconv.Atoi(issueidStr)
	if err != nil {
		http.Error(w, "invalid issue id", http.StatusBadRequest)
		return
	}
	var req struct {
		TagID int `json:"tag_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	uid, _ := users.UserIDFromContext(r.Context())
	if err = h.store.Attach(r.Context(), issueID, req.TagID, uid); err != nil {
		http.Error(w, "attach failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *Handlers) Detach(w http.ResponseWriter, r *http.Request) {
	issueIdStr := chi.URLParam(r, "issueId")
	issueID, _ := strconv.Atoi(issueIdStr)
	tagIdStr := chi.URLParam(r, "tagId")
	tagID, _ := strconv.Atoi(tagIdStr)

	uid, _ := users.UserIDFromContext(r.Context())
	if err := h.store.Detach(r.Context(), issueID, tagID, uid); err != nil {
		http.Error(w, "detach failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *Handlers) ListByIssue(w http.ResponseWriter, r *http.Request) {
	issueeIdStr := chi.URLParam(r, "issueId")
	issueID, _ := strconv.Atoi(issueeIdStr)
	uid, _ := users.UserIDFromContext(r.Context())
	list, err := h.store.ListByIssue(r.Context(), issueID, uid)
	if err != nil {
		http.Error(w, "list failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}
