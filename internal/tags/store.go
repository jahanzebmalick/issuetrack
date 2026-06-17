package tags

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Tag struct {
	ID        int    `json:"id"`
	ProjectID int    `json:"project_id"`
	Name      string `json:"name"`
	Color     string `json:"color"`
}
type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

func (s *Store) Create(ctx context.Context, projectID, ownerID int, name, color string) (Tag, error) {
	var tag Tag
	err := s.db.QueryRow(ctx, `
	INSERT INTO tags (project_id, name, color)
	SELECT $1, $3, $4 
	FROM projects
	WHERE id = $1 AND (
	owner_id = $2
	OR id IN (SELECT project_id FROM project_members WHERE user_id = $2)
	)
	RETURNING id, project_id, name, color`,
		projectID, ownerID, name, color,
	).Scan(&tag.ID, &tag.ProjectID, &tag.Name, &tag.Color)
	if err == pgx.ErrNoRows {
		return Tag{}, err
	}
	return tag, err
}
func (s *Store) ListByProject(ctx context.Context, projectID, ownerID int) ([]Tag, error) {
	rows, err := s.db.Query(ctx, `
	SELECT 
	id, 
	project_id,
	name,
	color
	FROM tags
	WHERE project_id = $1 AND project_id IN (
	SELECT id FROM projects WHERE owner_id = $2
	UNION
	SELECT project_id FROM project_members WHERE user_id = $2

	)`,
		projectID, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tags := []Tag{}
	for rows.Next() {
		var tag Tag
		err := rows.Scan(
			&tag.ID,
			&tag.ProjectID,
			&tag.Name,
			&tag.Color,
		)
		if err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}
func (s *Store) Delete(ctx context.Context, tagID, ownerID int) error {
	_, err := s.db.Exec(ctx, `
	DELETE FROM tags 
	WHERE id = $1 AND project_id IN (
	SELECT id FROM projects WHERE owner_id = $2
	UNION
	SELECT project_id FROM project_members WHERE user_id = $2
	)`, tagID, ownerID)
	return err
}
func (s *Store) Attach(ctx context.Context, issueID, tagID, ownerID int) error {
	_, err := s.db.Exec(ctx, `
	INSERT INTO issues_tags (issue_id, tag_id)
	SELECT $1, $2
	FROM issues i
	JOIN tags t ON t.project_id = i.project_id
	WHERE i.id = $1 AND t.id = $2
	  AND i.project_id IN (
	SELECT id FROM projects WHERE owner_id = $3
	UNION
	SELECT project_id FROM project_members WHERE user_id = $3
	)
	  `, issueID, tagID, ownerID)
	return err
}
func (s *Store) Detach(ctx context.Context, issueID, tagID, ownerID int) error {
	_, err := s.db.Exec(ctx, `
	DELETE FROM issues_tags
	WHERE issue_id = $1 AND tag_id = $2
	AND issue_id IN (
	SELECT id FROM issues WHERE project_id IN (
	SELECT id FROM projects WHERE owner_id = $3
	UNION
	SELECT project_id FROM project_members WHERE user_id = $3
	)
	)
	`, issueID, tagID, ownerID)
	return err
}
func (s *Store) ListByIssue(ctx context.Context, issueID, ownerID int) ([]Tag, error) {
	rows, err := s.db.Query(ctx, `
	SELECT t.id, t.project_id, t.name, t.color
	FROM tags t
	JOIN issues_tags it ON t.id = it.tag_id
	WHERE it.issue_id = $1
	AND t.project_id IN (
	SELECT id FROM projects WHERE owner_id = $2
	UNION
	SELECT project_id FROM project_members WHERE user_id = $2
	)
	`, issueID, ownerID)
	if err != nil {
		return []Tag{}, err
	}
	defer rows.Close()
	tags := []Tag{}
	for rows.Next() {
		var tag Tag
		err := rows.Scan(
			&tag.ID,
			&tag.ProjectID,
			&tag.Name,
			&tag.Color,
		)
		if err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}
