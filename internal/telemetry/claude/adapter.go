package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/shipproof/shipproof/internal/agent"
)

type claudeSession struct {
	PID                 int    `json:"pid"`
	SessionID           string `json:"sessionId"`
	CWD                 string `json:"cwd"`
	StartedAt           int64  `json:"startedAt"`
	Version             string `json:"version"`
	Status              string `json:"status"`
	UpdatedAt           int64  `json:"updatedAt"`
	StatusUpdatedAt     int64  `json:"statusUpdatedAt"`
	MessagingSocketPath string `json:"messagingSocketPath,omitempty"`
}

type claudeSettings struct {
	Model string `json:"model"`
}

type Adapter struct {
	sessionsDir string
}

func NewAdapter() *Adapter {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return &Adapter{
		sessionsDir: filepath.Join(home, ".claude", "sessions"),
	}
}

func NewAdapterWithDir(dir string) *Adapter {
	return &Adapter{sessionsDir: dir}
}

func (a *Adapter) Name() string {
	return "claude"
}

func (a *Adapter) Collect(projectDir string) (agent.AgentRun, error) {
	entries, err := os.ReadDir(a.sessionsDir)
	if err != nil {
		return agent.AgentRun{}, fmt.Errorf("read sessions directory: %w", err)
	}

	var mostRecent *claudeSession
	var latestTs int64

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(a.sessionsDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var session claudeSession
		if err := json.Unmarshal(data, &session); err != nil {
			continue
		}

		if session.CWD != projectDir {
			continue
		}

		if session.UpdatedAt > latestTs {
			latestTs = session.UpdatedAt
			mostRecent = &session
		}
	}

	if mostRecent == nil {
		return agent.AgentRun{}, fmt.Errorf("no session found for project %q", projectDir)
	}

	session := *mostRecent

	run := agent.AgentRun{
		Provider:     "claude-code",
		AgentVersion: session.Version,
		SessionID:    session.SessionID,
		StartedAt:    msToISO(session.StartedAt),
		EndedAt:      msToISO(session.UpdatedAt),
		ExitStatus:   session.Status,
	}

	model := readModel()
	if model != "" {
		run.Model = model
	}

	return run, nil
}

func readModel() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(home, ".claude", "settings.json"))
	if err != nil {
		return ""
	}
	var settings claudeSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return ""
	}
	return settings.Model
}

func msToISO(ms int64) string {
	if ms == 0 {
		return ""
	}
	return time.UnixMilli(ms).UTC().Format(time.RFC3339)
}
