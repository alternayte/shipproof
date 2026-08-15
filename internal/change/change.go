package change

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Record struct {
	SchemaVersion string `json:"schema_version"`
	ChangeID      string `json:"change_id"`
	SourcePath    string `json:"source_path"`
	SnapshotPath  string `json:"snapshot_path"`
	SHA256        string `json:"sha256"`
	ShapingRef    string `json:"shaping_ref,omitempty"`
	CapturedAt    string `json:"captured_at"`
}

func Path(root, changeID string) string {
	return filepath.Join(root, ".shipproof", "changes", changeID, "change.json")
}

func Start(root, changeID, sourcePath, shapingRef string) (Record, error) {
	if strings.TrimSpace(changeID) == "" {
		return Record{}, errors.New("change id is required")
	}

	recordPath := Path(root, changeID)
	if _, err := os.Stat(recordPath); err == nil {
		return Record{}, fmt.Errorf("change %q already exists", changeID)
	} else if !errors.Is(err, os.ErrNotExist) {
		return Record{}, fmt.Errorf("inspect change record: %w", err)
	}

	absSource, err := filepath.Abs(sourcePath)
	if err != nil {
		return Record{}, fmt.Errorf("resolve source path: %w", err)
	}

	content, err := os.ReadFile(absSource)
	if err != nil {
		return Record{}, fmt.Errorf("read source file: %w", err)
	}

	hash := sha256.Sum256(content)
	hashHex := hex.EncodeToString(hash[:])

	dir := filepath.Dir(recordPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Record{}, fmt.Errorf("create change directory: %w", err)
	}

	snapshotName := "snapshot" + filepath.Ext(sourcePath)
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
	relSource = filepath.ToSlash(relSource)

	relSnapshot, err := filepath.Rel(root, snapshotPath)
	if err != nil {
		relSnapshot = snapshotPath
	}
	relSnapshot = filepath.ToSlash(relSnapshot)

	record := Record{
		SchemaVersion: "0.1",
		ChangeID:      changeID,
		SourcePath:    relSource,
		SnapshotPath:  relSnapshot,
		SHA256:        hashHex,
		ShapingRef:    strings.TrimSpace(shapingRef),
		CapturedAt:    time.Now().UTC().Format(time.RFC3339),
	}

	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return Record{}, fmt.Errorf("encode change record: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(recordPath, data, 0o644); err != nil {
		return Record{}, fmt.Errorf("write change record: %w", err)
	}

	return record, nil
}

func Load(root, changeID string) (Record, error) {
	recordPath := Path(root, changeID)
	data, err := os.ReadFile(recordPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Record{}, fmt.Errorf("change %q not found", changeID)
		}
		return Record{}, fmt.Errorf("read change record: %w", err)
	}

	var record Record
	if err := json.Unmarshal(data, &record); err != nil {
		return Record{}, fmt.Errorf("parse change record: %w", err)
	}
	if err := record.Validate(); err != nil {
		return Record{}, err
	}
	return record, nil
}

func (record Record) Validate() error {
	if record.SchemaVersion != "0.1" {
		return fmt.Errorf("schema_version must be %q", "0.1")
	}
	if strings.TrimSpace(record.ChangeID) == "" {
		return errors.New("change_id is required")
	}
	if record.SHA256 == "" {
		return errors.New("sha256 is required")
	}
	if record.CapturedAt == "" {
		return errors.New("captured_at is required")
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

func HasVerificationPlan(root, changeID string) bool {
	planPath := filepath.Join(root, ".shipproof", "changes", changeID, "verification.json")
	_, err := os.Stat(planPath)
	return err == nil
}
