package domain

// MonitorEvent represents a single observable event in the agent workflow.
type MonitorEvent struct {
	ID      string `json:"id"`
	Time    string `json:"time"`
	Type    string `json:"type"`              // "lifecycle", "chat_response", "sensing_input", "chat_send", "tts"
	Summary string `json:"summary"`           // human-readable one-liner
	Detail  any    `json:"detail,omitempty"`   // structured payload for UI rendering
	RunID   string `json:"runId,omitempty"`    // OpenClaw run ID for grouping
	Phase   string `json:"phase,omitempty"`    // lifecycle phase if applicable
	State   string `json:"state,omitempty"`    // "partial", "final", etc.
	Error   string `json:"error,omitempty"`    // error message if any
}
