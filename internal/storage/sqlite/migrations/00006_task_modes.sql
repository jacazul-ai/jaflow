-- +goose Up
CREATE TABLE IF NOT EXISTS task_modes (
    code INTEGER PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);

INSERT OR IGNORE INTO task_modes (code, name) VALUES
    (0, 'UNSPECIFIED'),
    (1, 'DESIGN'),
    (2, 'SPIKE'),
    (3, 'INVESTIGATE'),
    (4, 'GUIDE'),
    (5, 'EXECUTE'),
    (6, 'REFINE'),
    (7, 'TEST'),
    (8, 'DEBUG'),
    (9, 'REVIEW');

ALTER TABLE tasks ADD COLUMN task_mode_code INTEGER
    REFERENCES task_modes(code);

UPDATE tasks
SET task_mode_code = CASE UPPER(TRIM(mode))
    WHEN 'DESIGN' THEN 1
    WHEN 'SPIKE' THEN 2
    WHEN 'INVESTIGATE' THEN 3
    WHEN 'GUIDE' THEN 4
    WHEN 'EXECUTE' THEN 5
    WHEN 'REFINE' THEN 6
    WHEN 'TEST' THEN 7
    WHEN 'DEBUG' THEN 8
    WHEN 'REVIEW' THEN 9
    ELSE 0
END;

CREATE INDEX IF NOT EXISTS tasks_mode_status_idx
    ON tasks (task_mode_code, status);

-- +goose Down
DROP INDEX IF EXISTS tasks_mode_status_idx;
-- SQLite keeps task_mode_code during rollback because dropping a column requires a table rebuild.
DROP TABLE IF EXISTS task_modes;
