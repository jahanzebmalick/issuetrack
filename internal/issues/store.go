package issues

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Issue struct {
	ID          int       `json:"id"`
	ProjectID   int       `json:"project_id"`
	CreatorID   int       `json:"creator_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Priority    string    `json:"priority"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}
func (s *Store) Create(ctx context.Context, projectID, creatorID int, title, desc string) (Issue, error) {
	var issue Issue
	err := s.db.QueryRow(ctx, `
        INSERT INTO issues (project_id, creator_id, title, description)
        SELECT $1, $2, $3, $4
        FROM projects WHERE id = $1 AND (
		 owner_id = $2
		 OR id IN (SELECT project_id FROM project_members WHERE user_id = $2)
		 )
        RETURNING id, project_id, creator_id, title, description, status, priority, created_at, updated_at
    `, projectID, creatorID, title, desc).Scan(
		&issue.ID, &issue.ProjectID, &issue.CreatorID,
		&issue.Title, &issue.Description, &issue.Status, &issue.Priority,
		&issue.CreatedAt, &issue.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return Issue{}, errors.New("project not found or not owned")
	}
	return issue, err
}

func (s *Store) ListByProject(ctx context.Context, projectID, ownerID int) ([]Issue, error) {
	rows, err := s.db.Query(ctx, `
	SELECT 
	id,
	project_id,
	creator_id, 
	title, 
	description, 
	status, 
	priority, 
	created_at, 
	updated_at
	FROM issues WHERE project_id = $1 AND project_id IN (
	SELECT id FROM projects WHERE owner_id = $2
	UNION
	SELECT project_id FROM project_members WHERE user_id = $2)`,
		projectID, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	issues := []Issue{}
	for rows.Next() {
		var issue Issue
		err := rows.Scan(
			&issue.ID,
			&issue.ProjectID,
			&issue.CreatorID,
			&issue.Title,
			&issue.Description,
			&issue.Status,
			&issue.Priority,
			&issue.CreatedAt,
			&issue.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		issues = append(issues, issue)
	}
	return issues, rows.Err()
}
func (s *Store) GetByID(ctx context.Context, issueID, ownerID int) (Issue, error) {
	var i Issue
	err := s.db.QueryRow(ctx, `
	SELECT id, project_id, creator_id, title, description, status,
	priority, created_at, updated_at From issues WHERE id = $1 AND project_id IN
	(SELECT id FROM projects WHERE owner_id = $2
	UNION
	SELECT project_id FROM project_members WHERE user_id = $2
	)`, issueID, ownerID).Scan(&i.ID, &i.ProjectID,
		&i.CreatorID, &i.Title, &i.Description, &i.Status, &i.Priority,
		&i.CreatedAt, &i.UpdatedAt)
	return i, err
}
func (s *Store) Update(ctx context.Context, issueID, ownerID int, title, desc, status, priority string) error {
	_, err := s.db.Exec(ctx, `
	UPDATE issues SET title = $1, description = $2, status = $3, priority = $4 WHERE id = $5 AND project_id IN
	(
	SELECT id FROM projects WHERE owner_id = $6
	UNION
	SELECT project_id FROM project_members WHERE user_id = $6
)`, title, desc, status, priority, issueID, ownerID)
	return err
}
func (s *Store) Delete(ctx context.Context, issueID, ownerID int) error {
	_, err := s.db.Exec(ctx, `
	DELETE FROM issues
	WHERE id = $1 AND project_id IN (
	SELECT id FROM projects WHERE owner_id = $2
	UNION
	SELECT project_id FROM project_members WHERE user_id = $2
)`, issueID, ownerID)
	return err
}
