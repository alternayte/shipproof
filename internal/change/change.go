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
	Ceremony      *int   `json:"ceremony,omitempty"`
	CapturedAt    string `json:"captured_at"`
}

// DefaultCeremony applies when a caller states no level, and when a record
// written before the field existed is loaded.
const DefaultCeremony = 1

// MaxCeremony is the highest level the triage-change skill can recommend.
const MaxCeremony = 3

// CeremonyLevel returns the recorded level. A record written before the field
// existed reads as DefaultCeremony. The field is a pointer because level 0 is
// a real value that must survive a round trip.
func (record Record) CeremonyLevel() int {
	if record.Ceremony == nil {
		return DefaultCeremony
	}
	return *record.Ceremony
}

func Path(root, changeID string) string {
	return filepath.Join(root, ".shipproof", "changes", changeID, "change.json")
}

func Start(root, changeID, sourcePath, shapingRef string, ceremony int) (Record, error) {
	return start(root, changeID, sourcePath, shapingRef, ceremony, false)
}

// Restart re-snapshots the source document of an existing change and rewrites
// the record. It exists because a stale intent has no other exit. A nil
// ceremony keeps the level that the record already carries. An empty shaping
// reference keeps the reference that the record already carries.
func Restart(root, changeID, sourcePath, shapingRef string, ceremony *int) (Record, error) {
	level := DefaultCeremony
	if ceremony != nil {
		level = *ceremony
	}

	previous, err := Load(root, changeID)
	if err == nil {
		if ceremony == nil {
			level = previous.CeremonyLevel()
		}
		if strings.TrimSpace(shapingRef) == "" {
			shapingRef = previous.ShapingRef
		}
	}

	return start(root, changeID, sourcePath, shapingRef, level, true)
}

func start(root, changeID, sourcePath, shapingRef string, ceremony int, force bool) (Record, error) {
	if ceremony < 0 || ceremony > MaxCeremony {
		return Record{}, fmt.Errorf("ceremony must be 0 to %d; got %d", MaxCeremony, ceremony)
	}
	if strings.TrimSpace(changeID) == "" {
		return Record{}, errors.New("change id is required")
	}

	recordPath := Path(root, changeID)
	if _, err := os.Stat(recordPath); err == nil {
		if !force {
			return Record{}, fmt.Errorf("change %q already exists", changeID)
		}
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
		Ceremony:      &ceremony,
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
	if record.Ceremony != nil && (*record.Ceremony < 0 || *record.Ceremony > MaxCeremony) {
		return fmt.Errorf("ceremony must be 0 to %d; got %d", MaxCeremony, *record.Ceremony)
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
// A stale record means the source intent changed after implementation began.
// A missing source document is stale with an empty current hash.
type Staleness struct {
	Stale        bool   `json:"stale"`
	SnapshotHash string `json:"snapshot_hash"`
	CurrentHash  string `json:"current_hash,omitempty"`
}

func (record Record) Staleness(root string) (Staleness, error) {
	sourcePath := filepath.Join(root, filepath.FromSlash(record.SourcePath))
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Staleness{Stale: true, SnapshotHash: record.SHA256}, nil
		}
		return Staleness{}, fmt.Errorf("read source file: %w", err)
	}

	hash := sha256.Sum256(content)
	current := hex.EncodeToString(hash[:])

	return Staleness{
		Stale:        current != record.SHA256,
		SnapshotHash: record.SHA256,
		CurrentHash:  current,
	}, nil
}

func HasVerificationPlan(root, changeID string) bool {
	planPath := filepath.Join(root, ".shipproof", "changes", changeID, "verification.json")
	_, err := os.Stat(planPath)
	return err == nil
}
