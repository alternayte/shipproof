package requirements

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// ErrUnconfirmed reports an attempt to write a foreign requirement set that no
// person confirmed. ShipProof never records a model extraction as a fact.
var ErrUnconfirmed = errors.New("a foreign requirement set needs a human confirmation")

// foreignObligation is the one documented pattern of decision D3. It matches a
// list item that states an obligation with MUST or SHALL.
//
// A specification tool often labels the item, as in `- **FR-001**: The system
// MUST reject the request`. The label is an identifier, not the obligation, so
// the pattern drops it and keeps the sentence.
//
// A heading is not an obligation. `## Why` and `## What Changes` are section
// titles, and proposing them as requirements would give a reader noise to
// prune rather than a set to judge.
var foreignObligation = regexp.MustCompile(`^\s*[-*]\s+(?:\*\*[^*]+\*\*:?\s*)?(.*\b(?:MUST|SHALL)\b.*)$`)

// IsNative reports whether the native adopter can read a document. A caller
// uses it to choose an adopter before it asks a person for anything.
func IsNative(changeID, sourcePath string) (bool, error) {
	body, err := os.ReadFile(sourcePath)
	if err != nil {
		return false, fmt.Errorf("read source document: %w", err)
	}
	if _, err := ParseNative(changeID, sourcePath, body); err != nil {
		if errors.Is(err, ErrNoNativeRequirement) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ProposeForeign extracts candidate requirements from any document. The result
// is a proposal. Every requirement carries human provenance and no
// confirmation stamp, so Save refuses it until a person confirms it.
func ProposeForeign(changeID, sourcePath string, body []byte) (Set, error) {
	changeID = strings.TrimSpace(changeID)
	if changeID == "" {
		return Set{}, errors.New("change_id is required")
	}

	set := Set{
		SchemaVersion: SchemaVersion,
		ChangeID:      changeID,
		Adopter:       AdopterForeign,
		SourcePath:    sourcePath,
	}

	inFence := false
	scanner := bufio.NewScanner(bytes.NewReader(body))
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}

		match := foreignObligation.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		statement := strings.TrimSpace(match[1])
		if statement == "" {
			continue
		}

		set.Requirements = append(set.Requirements, Requirement{
			ID:           fmt.Sprintf("%s-R%d", changeID, len(set.Requirements)+1),
			Statement:    statement,
			SourceAnchor: strings.TrimSpace(line),
			Provenance:   Human,
		})
	}
	if err := scanner.Err(); err != nil {
		return Set{}, fmt.Errorf("read document: %w", err)
	}
	if len(set.Requirements) == 0 {
		return Set{}, errors.New("the document states no obligation with MUST or SHALL")
	}
	if err := set.Validate(); err != nil {
		return Set{}, err
	}
	return set, nil
}

// RequiresConfirmation reports whether the set holds a human requirement with
// no confirmation stamp.
func (set Set) RequiresConfirmation() bool {
	for _, requirement := range set.Requirements {
		if requirement.Provenance == Human && strings.TrimSpace(requirement.ConfirmedAt) == "" {
			return true
		}
	}
	return false
}

// Confirm returns a copy of the set with a confirmation stamp on every human
// requirement. It never mutates the receiver.
func (set Set) Confirm(now time.Time) Set {
	stamped := set
	stamped.Requirements = make([]Requirement, len(set.Requirements))
	copy(stamped.Requirements, set.Requirements)
	moment := now.UTC().Format(time.RFC3339)
	for index := range stamped.Requirements {
		if stamped.Requirements[index].Provenance == Human {
			stamped.Requirements[index].ConfirmedAt = moment
		}
	}
	return stamped
}

// ProposalPath returns the location of a standing requirement proposal. A
// proposal lives beside the sidecar and never replaces it. A reader can open
// the file and judge what a person is about to confirm.
func ProposalPath(root, changeID string) string {
	return filepath.Join(root, ".shipproof", "changes", changeID, "requirements-proposal.json")
}

// SaveProposal writes a standing proposal. It refuses a set that needs no
// confirmation, because such a set belongs in the sidecar.
func SaveProposal(root string, set Set) (string, error) {
	if err := set.Validate(); err != nil {
		return "", err
	}
	if !set.RequiresConfirmation() {
		return "", errors.New("this set needs no confirmation; write it to the sidecar")
	}
	path := ProposalPath(root, set.ChangeID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create change directory: %w", err)
	}
	data, err := json.MarshalIndent(set, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return "", fmt.Errorf("write the proposal: %w", err)
	}
	return path, nil
}

// LoadProposal reads the standing proposal for one change.
func LoadProposal(root, changeID string) (Set, error) {
	data, err := os.ReadFile(ProposalPath(root, changeID))
	if err != nil {
		return Set{}, err
	}
	var set Set
	if err := json.Unmarshal(data, &set); err != nil {
		return Set{}, fmt.Errorf("read the proposal: %w", err)
	}
	return set, nil
}

// ClearProposal removes a standing proposal. A confirmed proposal has become
// the sidecar, so leaving it would let a reader confirm the same work twice.
func ClearProposal(root, changeID string) error {
	err := os.Remove(ProposalPath(root, changeID))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
