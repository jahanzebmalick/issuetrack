package projects

import (
	"encoding/json"
	"errors"
	"issuetrack/internal/users"
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
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	userID, _ := users.UserIDFromContext(r.Context())
	project, err := h.store.Create(r.Context(), userID, req.Name, req.Description)
	if err != nil {
		http.Error(w, "insert failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(project)
}
func (h *Handlers) ListMine(w http.ResponseWriter, r *http.Request) {
	uid, _ := users.UserIDFromContext(r.Context())

	list, err := h.store.ListMine(r.Context(), uid)
	if err != nil {
		http.Error(w, "query failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}
func (h *Handlers) Get(w http.ResponseWriter, r *http.Request) {
	idstr := chi.URLParam(r, "id")
	projectID, err := strconv.Atoi(idstr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	userID, _ := users.UserIDFromContext(r.Context())
	project, err := h.store.GetByID(r.Context(), projectID, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "lookup failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(project)
}
func (h *Handlers) Update(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	idstr := chi.URLParam(r, "id")
	projectID, err := strconv.Atoi(idstr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	userID, _ := users.UserIDFromContext(r.Context())
	if err = h.store.Update(r.Context(), projectID, userID, req.Name, req.Description); err != nil {
		http.Error(w, "update failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	idstr := chi.URLParam(r, "id")
	projectID, err := strconv.Atoi(idstr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	userID, _ := users.UserIDFromContext(r.Context())
	err = h.store.Delete(r.Context(), projectID, userID)
	if err != nil {
		http.Error(w, "delete failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
