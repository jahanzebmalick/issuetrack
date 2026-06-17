package attachments

import (
	"encoding/json"
	"io"
	"issuetrack/internal/db"
	"issuetrack/internal/users"
	"issuetrack/internal/ws"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handlers struct {
	store *Store
}

func NewHandlers(store *Store) *Handlers {
	return &Handlers{store: store}
}
func (h *Handlers) Upload(w http.ResponseWriter, r *http.Request) {
	issueIdStr := chi.URLParam(r, "issueId")
	issueID, _ := strconv.Atoi(issueIdStr)

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "form parse failed", http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	storageFilename := uuid.New().String() + filepath.Ext(header.Filename)
	storagePath := filepath.Join("storage", storageFilename)

	out, err := os.Create(storagePath)
	if err != nil {
		http.Error(w, "create failed", http.StatusInternalServerError)
		return
	}
	defer out.Close()

	size, err := io.Copy(out, file)
	if err != nil {
		os.Remove(storagePath)
		http.Error(w, "copy failed", http.StatusInternalServerError)
		return
	}
	mimeType := header.Header.Get("Content-Type")
	uid, _ := users.UserIDFromContext(r.Context())
	attachment, err := h.store.Create(r.Context(), issueID, uid, header.Filename, storagePath, size, mimeType)
	if err != nil {
		os.Remove(storagePath)
		http.Error(w, "save failed (access denied?)", http.StatusForbidden)
		return
	}
	var projectID int
	db.Pool.QueryRow(r.Context(), "SELECT project_id FROM issues WHERE id = $1", issueID).Scan(&projectID)
	if projectID > 0 {
		ws.GlobalHub.Publish(projectID, ws.Event{
			Type: "attachment_added",
			Data: attachment,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(attachment)
}
func (h *Handlers) ListByIssue(w http.ResponseWriter, r *http.Request) {
	issueIdStr := chi.URLParam(r, "issueId")
	issueID, _ := strconv.Atoi(issueIdStr)
	uid, _ := users.UserIDFromContext(r.Context())
	list, err := h.store.ListByIssue(r.Context(), issueID, uid)
	if err != nil {
		http.Error(w, "list failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}
func (h *Handlers) Download(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	attachmentID, _ := strconv.Atoi(idStr)

	uid, _ := users.UserIDFromContext(r.Context())
	a, err := h.store.GetByID(r.Context(), attachmentID, uid)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="`+a.Filename+`"`)
	if a.MimeType != "" {
		w.Header().Set("Content-Type", a.MimeType)
	}
	http.ServeFile(w, r, a.StoragePath)

}
func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	attachmentID, _ := strconv.Atoi(idStr)
	uid, _ := users.UserIDFromContext(r.Context())

	att, _ := h.store.GetByID(r.Context(), attachmentID, uid)
	var projectID int
	db.Pool.QueryRow(r.Context(), "SELECT project_id FROM issues WHERE id = $1", att.IssueID).Scan(&projectID)

	storagePath, err := h.store.Delete(r.Context(), attachmentID, uid)
	if err != nil {
		http.Error(w, "delete failed", http.StatusInternalServerError)
		return
	}
	_ = os.Remove(storagePath)
	if projectID > 0 {
		ws.GlobalHub.Publish(projectID, ws.Event{
			Type: "attachment_deleted",
			Data: map[string]int{"id": attachmentID, "issue_id": att.IssueID},
		})
	}
	w.WriteHeader(http.StatusNoContent)
}
