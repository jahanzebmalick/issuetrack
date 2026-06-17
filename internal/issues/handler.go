package issues

import (
	"encoding/json"
	"errors"
	"issuetrack/internal/activities"
	"issuetrack/internal/users"
	"issuetrack/internal/ws"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
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
	projIdStr := chi.URLParam(r, "projectId")
	projectID, err := strconv.Atoi(projIdStr)
	if err != nil {
		http.Error(w, "invalid project id", http.StatusBadRequest)
		return
	}
	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	uid, _ := users.UserIDFromContext(r.Context())
	issue, err := h.store.Create(
		r.Context(),
		projectID,
		uid,
		req.Title,
		req.Description,
	)
	if err != nil {
		http.Error(w, "not found", http.StatusInternalServerError)
		return
	}
	ws.GlobalHub.Publish(projectID, ws.Event{
		Type: "issue_created",
		Data: issue,
	})

	w.Header().Set("Content-Type", "application/json")
	log.Println("about to log activity")
	if err := activities.GlobalStore.Log(r.Context(), projectID, uid, "issue_created", &issue.ID, map[string]any{"title": issue.Title}); err != nil {
		log.Println("activity log failedd:", err)
	}
	json.NewEncoder(w).Encode(issue)
}
func (h *Handlers) ListByProject(w http.ResponseWriter, r *http.Request) {
	projIdStr := chi.URLParam(r, "projectId")
	projectID, err := strconv.Atoi(projIdStr)
	if err != nil {
		http.Error(w, "invalid project id", http.StatusBadRequest)
		return
	}
	uid, _ := users.UserIDFromContext(r.Context())
	list, err := h.store.ListByProject(r.Context(), projectID, uid)
	if err != nil {
		http.Error(w, "query failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)

}
func (h *Handlers) Get(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	issueID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid issue id", http.StatusBadRequest)
		return
	}
	uid, _ := users.UserIDFromContext(r.Context())
	issue, err := h.store.GetByID(r.Context(), issueID, uid)
	if errors.Is(err, pgx.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "query failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(issue)
}
func (h *Handlers) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	issueID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid issue id", http.StatusBadRequest)
		return
	}
	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Status      string `json:"status"`
		Priority    string `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	uid, _ := users.UserIDFromContext(r.Context())
	if err := h.store.Update(r.Context(), issueID, uid, req.Title, req.Description, req.Status, req.Priority); err != nil {
		http.Error(w, "update failed", http.StatusInternalServerError)
		return
	}
	issue, _ := h.store.GetByID(r.Context(), issueID, uid)
	ws.GlobalHub.Publish(issue.ProjectID, ws.Event{
		Type: "issue_updated",
		Data: issue,
	})
	activities.GlobalStore.Log(r.Context(), issue.ProjectID, uid, "issue_created", &issue.ID, map[string]any{"title": issue.Title, "status": issue.Status})
	w.WriteHeader(http.StatusNoContent)
}
func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	issueID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid issue id", http.StatusBadRequest)
		return
	}
	uid, _ := users.UserIDFromContext(r.Context())
	issue, _ := h.store.GetByID(r.Context(), issueID, uid)
	if err := h.store.Delete(r.Context(), issueID, uid); err != nil {
		http.Error(w, "delete failed", http.StatusInternalServerError)
		return
	}

	ws.GlobalHub.Publish(issue.ProjectID, ws.Event{
		Type: "issue_deleted",
		Data: map[string]int{"id": issueID},
	})
	activities.GlobalStore.Log(r.Context(), issue.ProjectID, uid, "issue_created", &issue.ID, map[string]any{"title": issue.Title})
	w.WriteHeader(http.StatusNoContent)

}
