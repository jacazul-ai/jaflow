package task

import "context"

// Status describes the current state of a workflow task.
type Status string

const (
	Pending   Status = "pending"
	Active    Status = "active"
	Completed Status = "completed"
)

// InitiativeStatus describes the lifecycle of an initiative.
type InitiativeStatus string

const (
	InitiativeActive    InitiativeStatus = "active"
	InitiativeBacklog   InitiativeStatus = "backlog"
	InitiativeCompleted InitiativeStatus = "completed"
	InitiativeArchived  InitiativeStatus = "archived"
)

// Initiative is a first-class workflow aggregate for a project.
type Initiative struct {
	ID             string
	ProjectID      string
	Name           string
	Status         InitiativeStatus
	ExternalTicket string
}

// CreateInitiativeInput contains the fields required to create or find an initiative.
type CreateInitiativeInput struct {
	ProjectID      string
	Name           string
	ExternalTicket string
}

// Task is the local workflow representation shared by task backends.
type Task struct {
	ID             string   `json:"id"`
	InitiativeID   string   `json:"initiative_id"`
	InitiativeName string   `json:"initiative_name"`
	Description    string   `json:"description"`
	Mode           string   `json:"mode,omitempty"`
	Status         Status   `json:"status"`
	Outcome        string   `json:"outcome,omitempty"`
	ExternalTicket string   `json:"external_ticket,omitempty"`
	StartedAt      string   `json:"started_at,omitempty"`
	CompletedAt    string   `json:"completed_at,omitempty"`
	Disposition    string   `json:"disposition,omitempty"`
	Dependencies   []string `json:"dependencies,omitempty"`

	// ProjectID and Plan are compatibility fields for the legacy adapter.
	ProjectID string `json:"project_id,omitempty"`
	Plan      string `json:"plan,omitempty"`
}

// FocusState stores the current navigation anchor for one project session.
type FocusState struct {
	ProjectID       string
	SessionID       string
	InitiativeID    string
	FocusedTaskID   string
	TaskStack       []FocusEntry
	PlansOfInterest []string
}

// SessionInfo describes one persisted project session.
type SessionInfo struct {
	ProjectID     string
	SessionID     string
	InitiativeID  string
	FocusedTaskID string
	UpdatedAt     string
}

// FocusEntry records one task in the focus stack.
type FocusEntry struct {
	TaskID       string `json:"task_id"`
	InitiativeID string `json:"initiative_id"`
}

// CreateTaskInput contains the fields required to create a workflow task.
type CreateTaskInput struct {
	InitiativeID string
	Description  string
	Mode         string
	Dependencies []string

	// ProjectID and Plan are compatibility fields for the legacy adapter.
	ProjectID string
	Plan      string
}

// CreateInput is the legacy task backend input shape.
type CreateInput = CreateTaskInput

// TaskBackend persists and retrieves workflow tasks.
type TaskBackend interface {
	Create(ctx context.Context, input CreateTaskInput) (Task, error)
	List(ctx context.Context, projectID string, plan string) ([]Task, error)
}
