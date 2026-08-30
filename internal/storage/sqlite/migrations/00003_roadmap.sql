-- +goose Up
CREATE TABLE IF NOT EXISTS roadmap_entries (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    initiative_id TEXT,
    phase TEXT NOT NULL,
    description TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE (project_id, description)
);

CREATE INDEX IF NOT EXISTS roadmap_project_idx
    ON roadmap_entries (project_id, phase, status);

-- +goose Down
DROP TABLE IF EXISTS roadmap_entries;
