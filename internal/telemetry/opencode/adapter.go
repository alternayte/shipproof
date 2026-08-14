package opencode

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/shipproof/shipproof/internal/agent"
)

type sessionRow struct {
	SessionID    string
	ProjectID    string
	Directory    string
	Title        string
	Version      string
	Cost         float64
	TokensInput  int64
	TokensOutput int64
	Model        string
	TimeCreated  int64
	TimeUpdated  int64
	Agent        string
}

type modelInfo struct {
	ID         string `json:"id"`
	ProviderID string `json:"providerID"`
}

type Adapter struct {
	dbPath string
}

func NewAdapter() *Adapter {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return &Adapter{
		dbPath: filepath.Join(home, ".local", "share", "opencode", "opencode.db"),
	}
}

func NewAdapterWithDB(dbPath string) *Adapter {
	return &Adapter{dbPath: dbPath}
}

func (a *Adapter) Name() string {
	return "opencode"
}

func (a *Adapter) Collect(projectDir string) (agent.AgentRun, error) {
	row, err := a.queryMostRecent(projectDir)
	if err != nil {
		return agent.AgentRun{}, err
	}

	run := agent.AgentRun{
		Provider:     "opencode",
		AgentVersion: row.Version,
		SessionID:    row.SessionID,
		StartedAt:    msToISO(row.TimeCreated),
		EndedAt:      msToISO(row.TimeUpdated),
		Cost:         row.Cost,
		Tokens:       &agent.TokenUsage{},
	}

	if row.Model != "" {
		var mi modelInfo
		if err := json.Unmarshal([]byte(row.Model), &mi); err == nil {
			run.Model = mi.ID
		}
	}

	if row.TokensInput > 0 || row.TokensOutput > 0 {
		run.Tokens.Input = row.TokensInput
		run.Tokens.Output = row.TokensOutput
	} else {
		run.Tokens = nil
	}

	return run, nil
}

func (a *Adapter) queryMostRecent(projectDir string) (sessionRow, error) {
	if _, err := os.Stat(a.dbPath); err != nil {
		return sessionRow{}, fmt.Errorf("opencode database not found at %q", a.dbPath)
	}

	query := fmt.Sprintf(
		"SELECT s.id, s.project_id, s.directory, s.title, s.version, s.cost, s.tokens_input, s.tokens_output, s.model, s.time_created, s.time_updated, s.agent FROM session s JOIN project p ON s.project_id = p.id WHERE p.worktree = '%s' ORDER BY s.time_updated DESC LIMIT 1",
		escapeSQL(projectDir),
	)

	output, err := exec.Command("sqlite3", "-separator", "|", a.dbPath, query).Output()
	if err != nil && len(output) == 0 {
		return sessionRow{}, fmt.Errorf("query opencode session: %w", err)
	}

	line := strings.TrimSpace(string(output))
	if line == "" {
		return sessionRow{}, fmt.Errorf("no session found for project %q", projectDir)
	}

	return parseRow(line), nil
}

func escapeSQL(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func parseRow(line string) sessionRow {
	fields := strings.Split(line, "|")
	row := sessionRow{}
	if len(fields) > 0 {
		row.SessionID = fields[0]
	}
	if len(fields) > 1 {
		row.ProjectID = fields[1]
	}
	if len(fields) > 2 {
		row.Directory = fields[2]
	}
	if len(fields) > 3 {
		row.Title = fields[3]
	}
	if len(fields) > 4 {
		row.Version = fields[4]
	}
	if len(fields) > 5 {
		fmt.Sscanf(fields[5], "%f", &row.Cost)
	}
	if len(fields) > 6 {
		fmt.Sscanf(fields[6], "%d", &row.TokensInput)
	}
	if len(fields) > 7 {
		fmt.Sscanf(fields[7], "%d", &row.TokensOutput)
	}
	if len(fields) > 8 {
		row.Model = fields[8]
	}
	if len(fields) > 9 {
		fmt.Sscanf(fields[9], "%d", &row.TimeCreated)
	}
	if len(fields) > 10 {
		fmt.Sscanf(fields[10], "%d", &row.TimeUpdated)
	}
	if len(fields) > 11 {
		row.Agent = fields[11]
	}
	return row
}

func msToISO(ms int64) string {
	if ms == 0 {
		return ""
	}
	return time.UnixMilli(ms).UTC().Format(time.RFC3339)
}
