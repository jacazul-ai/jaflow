-- +goose Up
ALTER TABLE tasks ADD COLUMN due_at TEXT NOT NULL DEFAULT '';

-- +goose Down
-- SQLite keeps the column during rollback because dropping it requires a table rebuild.
