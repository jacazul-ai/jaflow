-- +goose Up
CREATE TABLE IF NOT EXISTS initiatives (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    name TEXT NOT NULL,
    status TEXT NOT NULL,
    external_ticket TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE (project_id, name)
);

CREATE INDEX IF NOT EXISTS initiatives_project_idx
    ON initiatives (project_id, status, name);

CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    initiative_id TEXT NOT NULL REFERENCES initiatives(id) ON DELETE CASCADE,
    description TEXT NOT NULL,
    mode TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    outcome TEXT NOT NULL DEFAULT '',
    external_ticket TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS tasks_initiative_idx
    ON tasks (initiative_id, status, created_at);

CREATE TABLE IF NOT EXISTS task_dependencies (
    task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    depends_on_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    PRIMARY KEY (task_id, depends_on_id),
    CHECK (task_id <> depends_on_id)
);

CREATE INDEX IF NOT EXISTS task_dependencies_parent_idx
    ON task_dependencies (depends_on_id);

CREATE TABLE IF NOT EXISTS annotations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    body TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
    project_id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    focused_initiative_id TEXT,
    focused_task_id TEXT,
    task_stack_json TEXT NOT NULL DEFAULT '[]',
    updated_at TEXT NOT NULL,
    PRIMARY KEY (project_id, session_id)
);

CREATE TABLE IF NOT EXISTS cache_entries (
    project_id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    cache_key TEXT NOT NULL,
    output TEXT NOT NULL,
    output_hash TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (project_id, session_id, cache_key)
);

-- +goose Down
DROP TABLE IF EXISTS cache_entries;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS annotations;
DROP TABLE IF EXISTS task_dependencies;
DROP TABLE IF EXISTS tasks;
DROP TABLE IF EXISTS initiatives;
