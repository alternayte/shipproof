package repository

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Scope names one configuration file location.
type Scope string

const (
	ScopeLocal  Scope = "local"
	ScopeGlobal Scope = "global"
)

// DefaultRepairAttempts is the SDD Section 13.13 default bound.
const DefaultRepairAttempts = 2

var (
	// ErrKeyNotFound reports that a configuration key is absent in every scope.
	ErrKeyNotFound = errors.New("configuration key not found")
	// ErrCredentialKey reports a key that looks like a provider secret.
	// ShipProof is not a secret store.
	ErrCredentialKey = errors.New("ShipProof configuration must not hold a credential")

	keySegmentPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)
	credentialWords   = []string{"api_key", "apikey", "token", "secret", "password", "credential"}
)

// AgentConfig is the parsed `agent` section of a merged configuration.
type AgentConfig struct {
	Runner            string
	ReviewRunner      string
	RepairMaxAttempts int
	Runners           map[string]map[string]string
}

// LocalConfigPath returns the repository configuration file path.
func LocalConfigPath(root string) string {
	return filepath.Join(root, ".shipproof", "config.yaml")
}

// GlobalConfigPath returns the user configuration file path. It honours
// XDG_CONFIG_HOME and falls back to ~/.config.
func GlobalConfigPath() (string, error) {
	if base := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); base != "" {
		return filepath.Join(base, "shipproof", "config.yaml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".config", "shipproof", "config.yaml"), nil
}

// ConfigPath returns the file path for one scope.
func ConfigPath(root string, scope Scope) (string, error) {
	switch scope {
	case ScopeLocal:
		return LocalConfigPath(root), nil
	case ScopeGlobal:
		return GlobalConfigPath()
	default:
		return "", fmt.Errorf("scope must be local or global; got %q", scope)
	}
}

// ValidateKey rejects a malformed key and a key that looks like a credential.
func ValidateKey(key string) error {
	segments := strings.Split(strings.TrimSpace(key), ".")
	if len(segments) == 0 || segments[0] == "" {
		return errors.New("configuration key is required")
	}
	for _, segment := range segments {
		if !keySegmentPattern.MatchString(segment) {
			return fmt.Errorf("invalid configuration key segment %q", segment)
		}
		lower := strings.ToLower(strings.ReplaceAll(segment, "-", "_"))
		for _, word := range credentialWords {
			if strings.Contains(lower, word) {
				return fmt.Errorf("%w: key %q", ErrCredentialKey, key)
			}
		}
	}
	return nil
}

func loadTree(path string) (*yamlTree, error) {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &yamlTree{}, nil
		}
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()
	return parseTree(file)
}

// GetValue reads one key. The local scope wins over the global scope.
func GetValue(root, key string) (string, Scope, error) {
	if err := ValidateKey(key); err != nil {
		return "", "", err
	}
	path := strings.Split(key, ".")

	localTree, err := loadTree(LocalConfigPath(root))
	if err != nil {
		return "", "", err
	}
	if value, ok := localTree.get(path); ok {
		return value, ScopeLocal, nil
	}

	globalPath, err := GlobalConfigPath()
	if err != nil {
		return "", "", err
	}
	globalTree, err := loadTree(globalPath)
	if err != nil {
		return "", "", err
	}
	if value, ok := globalTree.get(path); ok {
		return value, ScopeGlobal, nil
	}

	return "", "", fmt.Errorf("%w: %s", ErrKeyNotFound, key)
}

// SetValue writes one key into one scope. It creates the file when needed and
// keeps existing keys, comments, and order.
func SetValue(root, key, value string, scope Scope) (string, error) {
	if err := ValidateKey(key); err != nil {
		return "", err
	}
	path, err := ConfigPath(root, scope)
	if err != nil {
		return "", err
	}
	tree, err := loadTree(path)
	if err != nil {
		return "", err
	}
	if err := tree.set(strings.Split(key, "."), value); err != nil {
		return "", fmt.Errorf("%s: cannot set %s, because %w. Edit the file by hand", path, key, err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create config directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(tree.render()), 0o644); err != nil {
		return "", fmt.Errorf("write config: %w", err)
	}
	return path, nil
}

// LoadAgentConfigScope reads the `agent` section from one scope only.
// Missing keys stay empty. RepairMaxAttempts is zero when the scope does not
// set it.
func LoadAgentConfigScope(root string, scope Scope) (AgentConfig, error) {
	config := AgentConfig{Runners: map[string]map[string]string{}}
	path, err := ConfigPath(root, scope)
	if err != nil {
		return config, err
	}
	tree, err := loadTree(path)
	if err != nil {
		return config, err
	}
	if err := applyTree(&config, tree); err != nil {
		return config, err
	}
	return config, nil
}

// LoadAgentConfig merges the global and the local `agent` section. Local keys
// win. Missing keys keep their documented defaults.
func LoadAgentConfig(root string) (AgentConfig, error) {
	config := AgentConfig{RepairMaxAttempts: DefaultRepairAttempts, Runners: map[string]map[string]string{}}

	for _, scope := range []Scope{ScopeGlobal, ScopeLocal} {
		path, err := ConfigPath(root, scope)
		if err != nil {
			return config, err
		}
		tree, err := loadTree(path)
		if err != nil {
			return config, err
		}
		if err := applyTree(&config, tree); err != nil {
			return config, err
		}
	}

	if config.RepairMaxAttempts == 0 {
		config.RepairMaxAttempts = DefaultRepairAttempts
	}
	return config, nil
}

func applyTree(config *AgentConfig, tree *yamlTree) error {
	if value, ok := tree.get([]string{"agent", "runner"}); ok && value != "" {
		config.Runner = value
	}
	if value, ok := tree.get([]string{"agent", "review_runner"}); ok && value != "" {
		config.ReviewRunner = value
	}
	if value, ok := tree.get([]string{"agent", "repair", "max_attempts"}); ok && value != "" {
		attempts, err := strconv.Atoi(value)
		if err != nil || attempts < 1 {
			return fmt.Errorf("agent.repair.max_attempts must be a positive integer; got %q", value)
		}
		config.RepairMaxAttempts = attempts
	}
	for _, name := range tree.keys([]string{"agent", "runners"}) {
		settings, ok := config.Runners[name]
		if !ok {
			settings = map[string]string{}
			config.Runners[name] = settings
		}
		for _, setting := range tree.keys([]string{"agent", "runners", name}) {
			if value, ok := tree.get([]string{"agent", "runners", name, setting}); ok {
				settings[setting] = value
			}
		}
	}
	return nil
}
