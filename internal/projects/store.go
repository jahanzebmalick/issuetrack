package projects

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Project struct {
	ID          int       `json:"id"`
	OwnerID     int       `json:"owner_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}
type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}
func (s *Store) Create(ctx context.Context, ownerID int, name string, description string) (Project, error) {
	var project Project
	err := s.db.QueryRow(ctx, `
	INSERT INTO projects (owner_id, name, description) VALUES ($1, $2, $3) RETURNING id, owner_id, name,
	description, created_at`, ownerID, name, description).Scan(&project.ID, &project.OwnerID, &project.Name, &project.Description, &project.CreatedAt)
	return project, err
}
func (s *Store) ListMine(ctx context.Context, ownerID int) ([]Project, error) {
	rows, err := s.db.Query(ctx, `
	SELECT id, owner_id, name, description, created_at FROM projects
	WHERE owner_id = $1
	OR id IN (SELECT project_id FROM project_members WHERE user_id = $1)`,
		ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	projects := []Project{}
	for rows.Next() {
		var p Project
		err := rows.Scan(
			&p.ID,
			&p.OwnerID,
			&p.Name,
			&p.Description,
			&p.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, nil
}
func (s *Store) GetByID(ctx context.Context, projectID, ownerID int) (Project, error) {
	var p Project
	err := s.db.QueryRow(ctx, `
	SELECT id, owner_id, name, description, created_at
	FROM projects
	WHERE id = $1 AND (
	 owner_id = $2
	 OR id IN (SELECT project_id FROm project_members WHERE user_id = $2))`,
		projectID, ownerID).Scan(&p.ID, &p.OwnerID, &p.Name,
		&p.Description, &p.CreatedAt,
	)
	return p, err
}
func (s *Store) Update(ctx context.Context, projectID int, ownerID int, name string, description string) error {
	_, err := s.db.Exec(ctx, `
	UPDATE projects SET name = $1, description = $2 WHERE id = $3 AND owner_id = $4`, name, description, projectID, ownerID)
	return err
}
func (s *Store) Delete(ctx context.Context, projectID int, ownerID int) error {
	_, err := s.db.Exec(ctx, `
	DELETE FROM projects WHERE id = $1 AND owner_id = $2`, projectID, ownerID)
	return err
}
