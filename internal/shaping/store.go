package shaping

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

type StartOptions struct {
	Kind    string
	Subject string
	ID      string
	Source  string
}

func Start(root string, options StartOptions) (Session, string, error) {
	id := strings.TrimSpace(options.ID)
	if id == "" {
		id = slugify(options.Subject)
	}
	if id == "" {
		return Session{}, "", errors.New("cannot derive session id; use --id")
	}
	if slugify(id) != id {
		return Session{}, "", fmt.Errorf("session id %q must use lowercase letters, digits, and hyphens", id)
	}

	session := Session{
		SchemaVersion: "0.1",
		SessionID:     id,
		Subject:       strings.TrimSpace(options.Subject),
		DocumentKind:  options.Kind,
		Source:        strings.TrimSpace(options.Source),
		State:         StateShaping,
		Decisions:     []Entry{},
		Assumptions:   []Entry{},
		Risks:         []Entry{},
		Unknowns:      []Entry{},
		Readiness: Readiness{
			Blockers:          []Entry{},
			DecisionsRequired: []Entry{},
		},
	}
	if err := session.Validate(); err != nil {
		return Session{}, "", err
	}

	path := Path(root, id)
	if _, err := os.Stat(path); err == nil {
		return Session{}, path, fmt.Errorf("shaping session %q already exists", id)
	} else if !errors.Is(err, os.ErrNotExist) {
		return Session{}, path, fmt.Errorf("inspect shaping session: %w", err)
	}
	if err := write(path, session); err != nil {
		return Session{}, path, err
	}
	return session, path, nil
}

func Load(root, id string) (Session, string, error) {
	path := Path(root, id)
	data, err := os.ReadFile(path)
	if err != nil {
		return Session{}, path, fmt.Errorf("read shaping session: %w", err)
	}
	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return Session{}, path, fmt.Errorf("parse shaping session: %w", err)
	}
	if err := session.Validate(); err != nil {
		return Session{}, path, fmt.Errorf("validate shaping session: %w", err)
	}
	return session, path, nil
}

func CheckFile(path string) (Session, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Session{}, fmt.Errorf("read shaping session: %w", err)
	}
	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return Session{}, fmt.Errorf("parse shaping session: %w", err)
	}
	if err := session.Validate(); err != nil {
		return Session{}, err
	}
	return session, nil
}

func Path(root, id string) string {
	return filepath.Join(root, ".shipproof", "shaping", id+".json")
}

func write(path string, session Session) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create shaping directory: %w", err)
	}
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("encode shaping session: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write shaping session: %w", err)
	}
	return nil
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = nonSlug.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	if len(value) > 64 {
		value = strings.Trim(value[:64], "-")
	}
	return value
}
