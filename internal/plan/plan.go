package plan

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var planIDPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// Record is the ShipProof plan record. A plan links approved intent to the
// changes it decomposes into.
type Record struct {
	SchemaVersion string   `json:"schema_version"`
	PlanID        string   `json:"plan_id"`
	SourcePath    string   `json:"source_path"`
	SnapshotPath  string   `json:"snapshot_path"`
	SHA256        string   `json:"sha256"`
	CreatedAt     string   `json:"created_at"`
	Changes       []string `json:"changes,omitempty"`
}

func Dir(root string) string {
	return filepath.Join(root, ".shipproof", "plans")
}

func Path(root, planID string) string {
	return filepath.Join(Dir(root), planID, "plan.json")
}

// Create snapshots a design document as a plan record.
func Create(root, sourcePath string) (Record, error) {
	absSource, err := filepath.Abs(sourcePath)
	if err != nil {
		return Record{}, fmt.Errorf("resolve source path: %w", err)
	}

	planID := planIDFromSource(absSource)
	if planID == "" {
		return Record{}, fmt.Errorf("cannot derive a plan id from %q; rename the file to use lowercase letters, digits, and hyphens", filepath.Base(sourcePath))
	}

	recordPath := Path(root, planID)
	if _, err := os.Stat(recordPath); err == nil {
		return Record{}, fmt.Errorf("plan %q already exists", planID)
	} else if !errors.Is(err, os.ErrNotExist) {
		return Record{}, fmt.Errorf("inspect plan record: %w", err)
	}

	content, err := os.ReadFile(absSource)
	if err != nil {
		return Record{}, fmt.Errorf("read source file: %w", err)
	}

	hash := sha256.Sum256(content)

	dir := filepath.Dir(recordPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Record{}, fmt.Errorf("create plan directory: %w", err)
	}

	snapshotName := "snapshot" + filepath.Ext(absSource)
	if snapshotName == "snapshot" {
		snapshotName = "snapshot.md"
	}
	snapshotPath := filepath.Join(dir, snapshotName)
	if err := os.WriteFile(snapshotPath, content, 0o644); err != nil {
		return Record{}, fmt.Errorf("write snapshot: %w", err)
	}

	relSource, err := filepath.Rel(root, absSource)
	if err != nil {
		relSource = absSource
	}
	relSnapshot, err := filepath.Rel(root, snapshotPath)
	if err != nil {
		relSnapshot = snapshotPath
	}

	record := Record{
		SchemaVersion: "0.1",
		PlanID:        planID,
		SourcePath:    filepath.ToSlash(relSource),
		SnapshotPath:  filepath.ToSlash(relSnapshot),
		SHA256:        hex.EncodeToString(hash[:]),
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
	}

	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return Record{}, fmt.Errorf("encode plan record: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(recordPath, data, 0o644); err != nil {
		return Record{}, fmt.Errorf("write plan record: %w", err)
	}

	return record, nil
}

func planIDFromSource(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	base = strings.ToLower(base)
	base = strings.ReplaceAll(base, "_", "-")
	base = strings.ReplaceAll(base, " ", "-")
	if !planIDPattern.MatchString(base) {
		return ""
	}
	return base
}

func Load(root, planID string) (Record, error) {
	recordPath := Path(root, planID)
	data, err := os.ReadFile(recordPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Record{}, fmt.Errorf("plan %q not found", planID)
		}
		return Record{}, fmt.Errorf("read plan record: %w", err)
	}

	var record Record
	if err := json.Unmarshal(data, &record); err != nil {
		return Record{}, fmt.Errorf("parse plan record: %w", err)
	}
	if err := record.Validate(); err != nil {
		return Record{}, err
	}
	return record, nil
}

// List returns all plan IDs under .shipproof/plans/.
func List(root string) ([]string, error) {
	entries, err := os.ReadDir(Dir(root))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read plans directory: %w", err)
	}

	var ids []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := Load(root, entry.Name()); err != nil {
			return nil, err
		}
		ids = append(ids, entry.Name())
	}
	return ids, nil
}

func (record Record) Validate() error {
	if record.SchemaVersion != "0.1" {
		return fmt.Errorf("schema_version must be %q", "0.1")
	}
	if !planIDPattern.MatchString(record.PlanID) {
		return errors.New("plan_id must use lowercase letters, digits, and hyphens")
	}
	if record.SourcePath == "" {
		return errors.New("source_path is required")
	}
	if record.SnapshotPath == "" {
		return errors.New("snapshot_path is required")
	}
	if record.SHA256 == "" {
		return errors.New("sha256 is required")
	}
	if record.CreatedAt == "" {
		return errors.New("created_at is required")
	}
	return nil
}

func (record Record) VerifyHash(root string) error {
	snapshotPath := filepath.Join(root, filepath.FromSlash(record.SnapshotPath))
	content, err := os.ReadFile(snapshotPath)
	if err != nil {
		return fmt.Errorf("read snapshot: %w", err)
	}

	hash := sha256.Sum256(content)
	hashHex := hex.EncodeToString(hash[:])
	if hashHex != record.SHA256 {
		return fmt.Errorf("snapshot hash mismatch: recorded %s, computed %s", record.SHA256, hashHex)
	}
	return nil
}

// Staleness compares the current source document with the recorded snapshot.
func (record Record) Staleness(root string) (bool, string, error) {
	sourcePath := filepath.Join(root, filepath.FromSlash(record.SourcePath))
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return true, "", nil
		}
		return false, "", fmt.Errorf("read source file: %w", err)
	}

	hash := sha256.Sum256(content)
	current := hex.EncodeToString(hash[:])
	return current != record.SHA256, current, nil
}
