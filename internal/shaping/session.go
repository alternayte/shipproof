package shaping

import (
	"errors"
	"fmt"
)

type State string

const (
	StateShaping              State = "shaping"
	StateBlocked              State = "blocked"
	StateReadyWithAssumptions State = "ready_with_assumptions"
	StateReady                State = "ready"
)

type Entry struct {
	ID      string `json:"id"`
	Summary string `json:"summary"`
	Source  string `json:"source,omitempty"`
}

type CurrentModel struct {
	Problem        string `json:"problem,omitempty"`
	DesiredOutcome string `json:"desired_outcome,omitempty"`
	Scope          string `json:"scope,omitempty"`
	Recommendation string `json:"recommendation,omitempty"`
}

type Readiness struct {
	Blockers          []Entry `json:"blockers"`
	DecisionsRequired []Entry `json:"decisions_required"`
}

type Session struct {
	SchemaVersion string       `json:"schema_version"`
	SessionID     string       `json:"session_id,omitempty"`
	Subject       string       `json:"subject"`
	DocumentKind  string       `json:"document_kind"`
	Source        string       `json:"source,omitempty"`
	State         State        `json:"state"`
	CurrentModel  CurrentModel `json:"current_model"`
	Decisions     []Entry      `json:"decisions"`
	Assumptions   []Entry      `json:"assumptions"`
	Risks         []Entry      `json:"risks"`
	Unknowns      []Entry      `json:"unknowns"`
	Readiness     Readiness    `json:"readiness"`
}

func (session Session) Validate() error {
	if session.SchemaVersion != "0.1" {
		return fmt.Errorf("schema_version must be %q", "0.1")
	}
	if session.Subject == "" {
		return errors.New("subject is required")
	}
	switch session.DocumentKind {
	case "prd", "sdd", "issue":
	default:
		return errors.New("document_kind must be prd, sdd, or issue")
	}
	switch session.State {
	case StateShaping, StateBlocked, StateReadyWithAssumptions, StateReady:
	default:
		return errors.New("state is invalid")
	}
	if err := validateEntries("decisions", session.Decisions); err != nil {
		return err
	}
	if err := validateEntries("assumptions", session.Assumptions); err != nil {
		return err
	}
	if err := validateEntries("risks", session.Risks); err != nil {
		return err
	}
	if err := validateEntries("unknowns", session.Unknowns); err != nil {
		return err
	}
	if err := validateEntries("readiness.blockers", session.Readiness.Blockers); err != nil {
		return err
	}
	if err := validateEntries("readiness.decisions_required", session.Readiness.DecisionsRequired); err != nil {
		return err
	}

	hasBlocking := len(session.Readiness.Blockers) > 0 || len(session.Readiness.DecisionsRequired) > 0
	hasAcceptedUncertainty := len(session.Assumptions) > 0 || len(session.Risks) > 0

	switch session.State {
	case StateBlocked:
		if !hasBlocking {
			return errors.New("blocked state requires a readiness blocker or required decision")
		}
	case StateReadyWithAssumptions:
		if hasBlocking {
			return errors.New("ready_with_assumptions cannot contain readiness blockers or required decisions")
		}
		if !hasAcceptedUncertainty {
			return errors.New("ready_with_assumptions requires at least one accepted assumption or risk")
		}
	case StateReady:
		if hasBlocking {
			return errors.New("ready state cannot contain readiness blockers or required decisions")
		}
		if hasAcceptedUncertainty {
			return errors.New("ready state cannot contain accepted assumptions or risks; use ready_with_assumptions")
		}
	}

	return nil
}

func validateEntries(field string, entries []Entry) error {
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if entry.ID == "" {
			return fmt.Errorf("%s entry id is required", field)
		}
		if entry.Summary == "" {
			return fmt.Errorf("%s entry %q summary is required", field, entry.ID)
		}
		if _, exists := seen[entry.ID]; exists {
			return fmt.Errorf("%s contains duplicate id %q", field, entry.ID)
		}
		seen[entry.ID] = struct{}{}
	}
	return nil
}
