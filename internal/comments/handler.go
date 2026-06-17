package comments

import (
	"encoding/json"
	"issuetrack/internal/activities"
	"issuetrack/internal/db"
	"issuetrack/internal/users"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type Handlers struct {
	store *Store
}

func NewHandlers(store *Store) *Handlers {
	return &Handlers{
		store: store,
	}
}
func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	issueidStr := chi.URLParam(r, "issueId")
	issueid, err := strconv.Atoi(issueidStr)
	if err != nil {
		http.Error(w, "invalid issue id", http.StatusBadRequest)
		return
	}
	var req struct {
		Body string `json:"body"`
	}
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	uid, _ := users.UserIDFromContext(r.Context())
	comment, err := h.store.Create(r.Context(), issueid, uid, req.Body)
	if err != nil {
		http.Error(w, "not found", http.StatusInternalServerError)
		return
	}
	var pid int
	db.Pool.QueryRow(r.Context(), "SELECT project_id FROM issues WHERE id = $1", comment.IssueID).Scan(&pid)

	w.Header().Set("Content-Type", "application/json")
	activities.GlobalStore.Log(r.Context(), pid, uid, "comment_added", &comment.ID, map[string]any{"issue_id": comment.IssueID})
	json.NewEncoder(w).Encode(comment)
}
func (h *Handlers) ListByIssue(w http.ResponseWriter, r *http.Request) {
	issueidStr := chi.URLParam(r, "issueId")
	issueid, err := strconv.Atoi(issueidStr)
	if err != nil {
		http.Error(w, "invalid issue id", http.StatusBadRequest)
		return
	}
	uid, _ := users.UserIDFromContext(r.Context())
	list, err := h.store.ListByIssue(r.Context(), issueid, uid)
	if err != nil {
		http.Error(w, "not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}
func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	idstr := chi.URLParam(r, "id")
	commentsID, err := strconv.Atoi(idstr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	uid, _ := users.UserIDFromContext(r.Context())
	if err := h.store.Delete(r.Context(), commentsID, uid); err != nil {
		http.Error(w, "delete failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
