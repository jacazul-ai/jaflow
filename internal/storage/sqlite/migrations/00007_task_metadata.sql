-- +goose Up
ALTER TABLE tasks ADD COLUMN priority TEXT NOT NULL DEFAULT 'M';
ALTER TABLE tasks ADD COLUMN urgency REAL NOT NULL DEFAULT 0;
ALTER TABLE tasks ADD COLUMN wait_until TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS tasks_priority_urgency_idx
    ON tasks (priority, urgency, status);

-- +goose Down
DROP INDEX IF EXISTS tasks_priority_urgency_idx;
-- SQLite keeps the metadata columns during rollback because dropping columns
-- requires rebuilding the tasks table.
