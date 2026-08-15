package agent

type TokenUsage struct {
	Input  int64 `json:"input,omitempty"`
	Output int64 `json:"output,omitempty"`
}

type AgentRun struct {
	Provider      string      `json:"provider,omitempty"`
	AgentVersion  string      `json:"agent_version,omitempty"`
	Model         string      `json:"model,omitempty"`
	StartedAt     string      `json:"started_at,omitempty"`
	EndedAt       string      `json:"ended_at,omitempty"`
	SessionID     string      `json:"session_id,omitempty"`
	Cost          float64     `json:"cost,omitempty"`
	Tokens        *TokenUsage `json:"tokens,omitempty"`
	ToolCallCount int64       `json:"tool_call_count,omitempty"`
	ExitStatus    string      `json:"exit_status,omitempty"`
	RawLogRef     string      `json:"raw_log_ref,omitempty"`
}

type Adapter interface {
	Name() string
	Collect(projectDir string) (AgentRun, error)
}

// RawLogProvider is implemented by adapters that can locate the raw session
// transcript for the most recent run. Capture levels redacted and full store
// this transcript under .shipproof/runs/<change-id>/agent-raw/.
type RawLogProvider interface {
	RawLogPath(projectDir string) (string, error)
}
