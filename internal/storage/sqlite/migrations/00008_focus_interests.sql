-- +goose Up
ALTER TABLE sessions ADD COLUMN plans_of_interest_json TEXT NOT NULL DEFAULT '[]';

-- +goose Down
-- SQLite keeps the column during rollback because dropping columns requires
-- rebuilding the sessions table.
