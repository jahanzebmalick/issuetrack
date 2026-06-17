CREATE TABLE IF NOT EXISTS activities (
    id SERIAL PRIMARY KEY,
    project_id INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    actor_id INTEGER NOT NULL REFERENCES users(id),
    kind TEXT NOT NULL,
    target_id INTEGER,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_activities_project ON activities(project_id, created_at DESC);