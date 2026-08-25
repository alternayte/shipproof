package requirements

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// ErrUnconfirmed reports an attempt to write a foreign requirement set that no
// person confirmed. ShipProof never records a model extraction as a fact.
var ErrUnconfirmed = errors.New("a foreign requirement set needs a human confirmation")

// foreignHeading matches a markdown heading of level two or deeper. A level
// one heading is the document title, not a requirement.
var foreignHeading = regexp.MustCompile(`^(#{2,6})\s+(.+?)\s*$`)

// foreignObligation matches a list item that states an obligation.
var foreignObligation = regexp.MustCompile(`^\s*[-*]\s+((?:MUST|SHALL)\b.*)$`)

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

		statement := ""
		if match := foreignHeading.FindStringSubmatch(line); match != nil {
			statement = strings.TrimSpace(match[2])
		} else if match := foreignObligation.FindStringSubmatch(line); match != nil {
			statement = strings.TrimSpace(match[1])
		}
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
		return Set{}, errors.New("the document holds no candidate requirement")
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
