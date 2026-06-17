package activities

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Activity struct {
	ID        int             `json:"id"`
	ProjectID int             `json:"project_id"`
	ActorID   int             `json:"actor_id"`
	Username  string          `json:"username"`
	Kind      string          `json:"kind"`
	TargetID  *int            `json:"target_id"`
	Metadata  json.RawMessage `json:"metadata"`
	CreatedAt time.Time       `json:"created_at"`
}
type Store struct {
	db *pgxpool.Pool
}

var GlobalStore *Store

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}
func (s *Store) Log(ctx context.Context, projectID, actorID int, kind string, targetID *int, metadata any) error {
	var meta []byte
	if metadata != nil {
		b, err := json.Marshal(metadata)
		if err != nil {
			return err
		}
		meta = b
	}
	_, err := s.db.Exec(ctx, `
	INSERT INTO activities (project_id, actor_id, kind, target_id, metadata)
	VALUES ($1, $2, $3, $4, $5)
	`, projectID, actorID, kind, targetID, meta)
	return err
}
func (s *Store) ListByProject(ctx context.Context, projectID, userID, limit int) ([]Activity, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.db.Query(ctx, `
	SELECT a.id, a.project_id, a.actor_id, u.username, a.kind, a.target_id, a.metadata, a.created_at
	FROM activities a 
	JOIN users u ON u.id = a.actor_id
	WHERE a.project_id = $1 AND a.project_id IN (
	SELECT id FROM projects WHERE owner_id = $2
	UNION
	SELECT project_id FROM project_members WHERE user_id = $2
	)
	ORDER BY a.created_at DESC
	LIMIT $3`, projectID, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []Activity{}
	for rows.Next() {
		var a Activity
		if err := rows.Scan(
			&a.ID,
			&a.ProjectID,
			&a.ActorID,
			&a.Username,
			&a.Kind,
			&a.TargetID,
			&a.Metadata,
			&a.CreatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}
