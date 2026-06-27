package main

// EventType represents the type of agent event.
type EventType string

const (
	EventFileRead   EventType = "file_read"
	EventFileWrite  EventType = "file_write"
	EventFileDelete EventType = "file_delete"
	EventCommand    EventType = "command"
	EventThought    EventType = "thought"
	EventPlanStep   EventType = "plan_step"
	EventError      EventType = "error"
	EventDone       EventType = "done"
)

// Event is a structured event emitted by the parser and sent to the dashboard.
type Event struct {
	Type      EventType `json:"type"`
	Timestamp float64   `json:"timestamp"` // seconds since start
	Payload   string    `json:"payload"`   // file path, command text, thought text, etc.
	SessionID string    `json:"session_id"`
}

// Node represents a file or concept node in the knowledge graph.
type Node struct {
	ID        string  `json:"id"`
	Label     string  `json:"label"`
	Kind      string  `json:"kind"` // "file", "command", "thought"
	EventType string  `json:"event_type"`
	X         float64 `json:"x,omitempty"`
	Y         float64 `json:"y,omitempty"`
}

// Edge represents a relationship between two nodes.
type Edge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Kind   string `json:"kind"` // "read", "write", "exec"
}
