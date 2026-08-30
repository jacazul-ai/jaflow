package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jacazul-ai/jaflow/internal/task"
)

// FindInitiative returns one initiative by project and name.
func (s *Store) FindInitiative(ctx context.Context, projectID string, name string) (task.Initiative, error) {
	if projectID == "" || name == "" {
		return task.Initiative{}, errors.New("project ID and initiative name are required")
	}
	initiative, err := s.findInitiative(ctx, projectID, name)
	if errors.Is(err, sql.ErrNoRows) {
		return task.Initiative{}, fmt.Errorf("initiative %q not found", name)
	}
	if err != nil {
		return task.Initiative{}, fmt.Errorf("find initiative: %w", err)
	}
	return initiative, nil
}

// LoadFocus returns session focus or an empty state when no anchor exists.
func (s *Store) LoadFocus(ctx context.Context, projectID string, sessionID string) (task.FocusState, error) {
	if projectID == "" || sessionID == "" {
		return task.FocusState{}, errors.New("project ID and session ID are required")
	}

	initiativeID, focusedTaskID, stackJSON, err := s.loadFocusRow(ctx, projectID, sessionID)
	if errors.Is(err, sql.ErrNoRows) && sessionID != "global" {
		initiativeID, focusedTaskID, stackJSON, err = s.loadFocusRow(ctx, projectID, "global")
	}
	if errors.Is(err, sql.ErrNoRows) {
		return task.FocusState{
			ProjectID: projectID,
			SessionID: sessionID,
			TaskStack: []task.FocusEntry{},
		}, nil
	}
	if err != nil {
		return task.FocusState{}, fmt.Errorf("load focus: %w", err)
	}

	state := task.FocusState{
		ProjectID:     projectID,
		SessionID:     sessionID,
		InitiativeID:  initiativeID,
		FocusedTaskID: focusedTaskID,
		TaskStack:     []task.FocusEntry{},
	}
	if err := json.Unmarshal([]byte(stackJSON), &state.TaskStack); err != nil {
		return task.FocusState{}, fmt.Errorf("decode focus stack: %w", err)
	}
	if state.TaskStack == nil {
		state.TaskStack = []task.FocusEntry{}
	}
	return state, nil
}

func (s *Store) loadFocusRow(ctx context.Context, projectID string, sessionID string) (string, string, string, error) {
	var initiativeID string
	var focusedTaskID string
	var stackJSON string
	err := s.db.QueryRowContext(ctx, `
		SELECT focused_initiative_id, focused_task_id, task_stack_json
		FROM sessions
		WHERE project_id = ? AND session_id = ?
	`, projectID, sessionID).Scan(&initiativeID, &focusedTaskID, &stackJSON)
	return initiativeID, focusedTaskID, stackJSON, err
}

// SaveFocus persists the project session anchor and task stack.
func (s *Store) SaveFocus(ctx context.Context, state task.FocusState) error {
	if state.ProjectID == "" || state.SessionID == "" {
		return errors.New("project ID and session ID are required")
	}
	stackJSON, err := json.Marshal(state.TaskStack)
	if err != nil {
		return fmt.Errorf("encode focus stack: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO sessions
			(project_id, session_id, focused_initiative_id, focused_task_id,
			 task_stack_json, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (project_id, session_id) DO UPDATE SET
			focused_initiative_id = excluded.focused_initiative_id,
			focused_task_id = excluded.focused_task_id,
			task_stack_json = excluded.task_stack_json,
			updated_at = excluded.updated_at
	`, state.ProjectID, state.SessionID, state.InitiativeID, state.FocusedTaskID,
		string(stackJSON), timestamp())
	if err != nil {
		return fmt.Errorf("save focus: %w", err)
	}
	return nil
}

// ListSessions returns persisted sessions for one project.
func (s *Store) ListSessions(ctx context.Context, projectID string) ([]task.SessionInfo, error) {
	if projectID == "" {
		return nil, errors.New("project ID is required")
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT project_id, session_id, focused_initiative_id,
		       focused_task_id, updated_at
		FROM sessions
		WHERE project_id = ?
		ORDER BY updated_at DESC, session_id
	`, projectID)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []task.SessionInfo
	for rows.Next() {
		var current task.SessionInfo
		if err := rows.Scan(
			&current.ProjectID,
			&current.SessionID,
			&current.InitiativeID,
			&current.FocusedTaskID,
			&current.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		sessions = append(sessions, current)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sessions: %w", err)
	}
	return sessions, nil
}

// AddAnnotation appends a canonical structured note to a task.
func (s *Store) AddAnnotation(ctx context.Context, taskID string, kind string, body string) error {
	current, err := s.GetTask(ctx, taskID)
	if err != nil {
		return err
	}
	canonicalKind, ok := task.NormalizeAnnotationKind(kind)
	if !ok {
		return fmt.Errorf("invalid annotation kind %q", strings.TrimSpace(kind))
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return errors.New("annotation body is required")
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO annotations (task_id, kind, body, created_at)
		VALUES (?, ?, ?, ?)
	`, current.ID, canonicalKind, body, timestamp())
	if err != nil {
		return fmt.Errorf("add annotation: %w", err)
	}
	return nil
}

// SetCache stores a session-scoped derived output until expiresAt.
func (s *Store) SetCache(ctx context.Context, projectID string, sessionID string, key string, output string, expiresAt time.Time) error {
	if projectID == "" || sessionID == "" || key == "" {
		return errors.New("project ID, session ID, and cache key are required")
	}
	hash := sha256.Sum256([]byte(output))
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO cache_entries
			(project_id, session_id, cache_key, output, output_hash, expires_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (project_id, session_id, cache_key) DO UPDATE SET
			output = excluded.output,
			output_hash = excluded.output_hash,
			expires_at = excluded.expires_at,
			updated_at = excluded.updated_at
	`, projectID, sessionID, key, output, hex.EncodeToString(hash[:]),
		expiresAt.UTC().Format(time.RFC3339Nano), timestamp())
	if err != nil {
		return fmt.Errorf("set cache: %w", err)
	}
	return nil
}

// GetCache returns a non-expired cached output.
func (s *Store) GetCache(ctx context.Context, projectID string, sessionID string, key string, now time.Time) (string, bool, error) {
	var output string
	var expiresAt string
	err := s.db.QueryRowContext(ctx, `
		SELECT output, expires_at
		FROM cache_entries
		WHERE project_id = ? AND session_id = ? AND cache_key = ?
	`, projectID, sessionID, key).Scan(&output, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("get cache: %w", err)
	}
	expires, err := time.Parse(time.RFC3339Nano, expiresAt)
	if err != nil {
		return "", false, fmt.Errorf("parse cache expiry: %w", err)
	}
	if !now.Before(expires) {
		return "", false, nil
	}
	return output, true, nil
}

// ClearCache removes cache entries matching a key prefix for one session.
func (s *Store) ClearCache(ctx context.Context, projectID string, sessionID string, prefix string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM cache_entries
		WHERE project_id = ? AND session_id = ? AND cache_key LIKE ?
	`, projectID, sessionID, prefix+"%")
	if err != nil {
		return fmt.Errorf("clear cache: %w", err)
	}
	return nil
}

// ClearCacheKey removes one exact cache entry for a session.
func (s *Store) ClearCacheKey(ctx context.Context, projectID string, sessionID string, key string) error {
	if projectID == "" || sessionID == "" || key == "" {
		return errors.New("project ID, session ID, and cache key are required")
	}
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM cache_entries
		WHERE project_id = ? AND session_id = ? AND cache_key = ?
	`, projectID, sessionID, key)
	if err != nil {
		return fmt.Errorf("clear cache key: %w", err)
	}
	return nil
}
