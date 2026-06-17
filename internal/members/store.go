package members

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Member struct {
	ProjectID int       `json:"project_id"`
	UserID    int       `json:"user_id"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	JoinedAT  time.Time `json:"joined_at"`
}
type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}
func (s *Store) Invite(ctx context.Context, projectID, ownerID int, username, role string) (Member, error) {
	var member Member
	err := s.db.QueryRow(ctx, `
	INSERT INTO project_members (project_id, user_id, role)
	SELECT p.id, u.id, $4
	FROM projects p, users u
	WHERE p.id = $1 AND u.username = $3 AND p.owner_id = $2
	RETURNING project_id, user_id, role, joined_at`, projectID, ownerID, username, role).Scan(
		&member.ProjectID,
		&member.UserID,
		&member.Role,
		&member.JoinedAT,
	)
	if err != nil {
		return Member{}, err
	}
	return member, err
}
func (s *Store) ListByProject(ctx context.Context, projectID, ownerID int) ([]Member, error) {
	rows, err := s.db.Query(ctx, `
	SELECT 
	pm.project_id,
	pm.user_id,
	u.username, 
	pm.role, 
	pm.joined_at
	FROM project_members pm
	JOIN users u ON u.id = pm.user_id
	WHERE pm.project_id = $1
	 AND pm.project_id IN (SELECT id FROM projects WHERE owner_id = $2)`, projectID, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	members := []Member{}
	for rows.Next() {
		var member Member
		err := rows.Scan(
			&member.ProjectID,
			&member.UserID,
			&member.Username,
			&member.Role,
			&member.JoinedAT,
		)
		if err != nil {
			return nil, err
		}
		members = append(members, member)
	}
	return members, rows.Err()
}
func (s *Store) UpdateRole(ctx context.Context, projectID, userID, ownerID int, role string) error {
	_, err := s.db.Exec(ctx, `
	UPDATE project_members SET role = $1 WHERE project_id = $2 AND user_id = $3
	AND project_id IN (SELECT id FROM projects WHERE owner_id = $4)`, role, projectID, userID, ownerID)
	return err
}
func (s *Store) Remove(ctx context.Context, projectID, userID, ownerID int) error {
	_, err := s.db.Exec(ctx, `
	DELETE FROM project_members WHERE project_id = $1 AND user_id = $2
	AND project_id IN (SELECT id FROM projects WHERE owner_id = $3)`, projectID, userID, ownerID)
	return err
}
