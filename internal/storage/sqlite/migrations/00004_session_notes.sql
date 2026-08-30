-- +goose Up
CREATE TABLE IF NOT EXISTS session_notes (
    project_id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    content TEXT NOT NULL,
    acknowledged_at TEXT NOT NULL DEFAULT '',
    updated_at TEXT NOT NULL,
    PRIMARY KEY (project_id, session_id)
);

-- +goose Down
DROP TABLE IF EXISTS session_notes;
