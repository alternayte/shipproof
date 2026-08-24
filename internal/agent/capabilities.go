package agent

// RunnerCapabilities describes what a runner supports. ShipProof probes
// capabilities. It must not assume feature parity between runners.
// Capabilities must not leak into the Change or Evidence models.
type RunnerCapabilities struct {
	Resume           bool `json:"resume"`
	ReadOnly         bool `json:"read_only"`
	WorkspaceWrite   bool `json:"workspace_write"`
	StructuredOutput bool `json:"structured_output"`
	Streaming        bool `json:"streaming"`
}

// RunnerStatus is the probe result for one runner.
type RunnerStatus struct {
	Installed     bool               `json:"installed"`
	Authenticated bool               `json:"authenticated"`
	Version       string             `json:"version"`
	Capabilities  RunnerCapabilities `json:"capabilities"`
	Detail        string             `json:"detail"`
}

// Usable reports whether the runner can execute work now.
func (status RunnerStatus) Usable() bool {
	return status.Installed && status.Authenticated
}
