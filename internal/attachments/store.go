package attachments

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Attachment struct {
	ID          int       `json:"id"`
	IssueID     int       `json:"issue_id"`
	UploaderID  int       `json:"uploader_id"`
	Filename    string    `json:"filename"`
	StoragePath string    `json:"-"`
	SizeBytes   int64     `json:"size_bytes"`
	MimeType    string    `json:"mime_type"`
	CreatedAt   time.Time `json:"created_at"`
}
type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

func (s *Store) Create(ctx context.Context, issueID, uploaderID int, filename, storagePath string, sizeBytes int64, mimeType string) (Attachment, error) {
	var a Attachment
	err := s.db.QueryRow(ctx, `
	INSERT INTO attachments (issue_id, uploader_id,
	filename, storage_path, size_bytes, mime_type)
	SELECT $1, $2, $3, $4, $5, $6
	FROM issues 
	WHERE id = $1 AND project_id IN (
	SELECT id FROM projects WHERE owner_id = $2
	UNION
	SELECT project_id FROM project_members WHERE user_id = $2
	)
	RETURNING id, 
	issue_id, uploader_id, filename, storage_path,
	size_bytes, mime_type, created_at`,
		issueID, uploaderID, filename, storagePath, sizeBytes, mimeType).Scan(
		&a.ID,
		&a.IssueID,
		&a.UploaderID,
		&a.Filename,
		&a.StoragePath,
		&a.SizeBytes,
		&a.MimeType,
		&a.CreatedAt,
	)
	return a, err
}
func (s *Store) ListByIssue(ctx context.Context, issueID, userID int) ([]Attachment, error) {
	rows, err := s.db.Query(ctx, `
	SELECT id, issue_id, uploader_id, filename, storage_path, size_bytes, mime_type, created_at
	FROM attachments
	WHERE issue_id = $1 AND issue_id IN (
	SELECT id FROM issues WHERE project_id IN (
	SELECT id FROM projects WHERE owner_id = $2
	UNION
	SELECT project_id FROM project_members WHERE user_id = $2
	)
	)
	ORDER BY created_at ASC`, issueID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []Attachment{}
	for rows.Next() {
		var a Attachment
		if err := rows.Scan(
			&a.ID,
			&a.IssueID,
			&a.UploaderID,
			&a.Filename,
			&a.StoragePath,
			&a.SizeBytes,
			&a.MimeType,
			&a.CreatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}
func (s *Store) GetByID(ctx context.Context, attachmentID, userID int) (Attachment, error) {
	var a Attachment
	err := s.db.QueryRow(ctx, `
	SELECT a.id, a.issue_id, a.uploader_id, a.filename, a.storage_path, a.size_bytes, a.mime_type, a.created_at
	FROM attachments a
	JOIN issues i ON i.id = a.issue_id
	WHERE a.id = $1 AND i.project_id IN (
	SELECT id FROM projects WHERE owner_id = $2
	UNION
	SELECT project_id FROM project_members WHERE user_id = $2
	
	)`,
		attachmentID, userID).Scan(
		&a.ID, &a.IssueID, &a.UploaderID, &a.Filename, &a.StoragePath,
		&a.SizeBytes, &a.MimeType, &a.CreatedAt,
	)
	return a, err

}
func (s *Store) Delete(ctx context.Context, attachmentID, uploaderID int) (string, error) {
	var storagePath string
	err := s.db.QueryRow(ctx, `
	DELETE FROM attachments
	WHERE id = $1 AND uploader_id = $2
	RETURNING storage_path
	`, attachmentID, uploaderID).Scan(&storagePath)
	return storagePath, err
}
