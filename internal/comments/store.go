package comments

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Comment struct {
	ID        int       `json:"id"`
	IssueID   int       `json:"issue_id"`
	AuthorID  int       `json:"author_id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}
func (s *Store) Create(ctx context.Context, issueID, authorID int, body string) (Comment, error) {
	var comment Comment
	err := s.db.QueryRow(ctx, `
	INSERT INTO comments (issue_id, author_id, body) 
	SELECT $1, $2, $3 
	FROM issues 
	WHERE id = $1 AND project_id IN (
	SELECT id FROM projects WHERE owner_id = $2
	UNION
	SELECT project_id FROM project_members WHERE user_id = $2
	)
	RETURNING id, issue_id, author_id, body, created_at`, issueID, authorID, body).Scan(&comment.ID,
		&comment.IssueID, &comment.AuthorID, &comment.Body, &comment.CreatedAt)
	if err == pgx.ErrNoRows {
		return Comment{}, err
	}
	return comment, err
}
func (s *Store) ListByIssue(ctx context.Context, issueID, ownerID int) ([]Comment, error) {
	rows, err := s.db.Query(ctx, `
	SELECT
	id,
	issue_id,
	author_id,
	body,
	created_at
	FROM comments
	WHERE issue_id = $1 AND issue_id IN (
	SELECT id FROM issues
	WHERE project_id IN (
	SELECT id FROM projects WHERE owner_id = $2
	UNION
	SELECT project_id FROM project_members WHERE user_id = $2
	)
	)`, issueID, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	comments := []Comment{}
	for rows.Next() {
		var comment Comment
		err := rows.Scan(
			&comment.ID,
			&comment.IssueID,
			&comment.AuthorID,
			&comment.Body,
			&comment.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		comments = append(comments, comment)
	}
	return comments, rows.Err()
}
func (s *Store) Delete(ctx context.Context, commentID, authorID int) error {
	_, err := s.db.Exec(ctx, `
	DELETE FROM comments
	WHERE id = $1 AND author_id = $2`,
		commentID,
		authorID,
	)
	return err

}
