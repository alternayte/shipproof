package requirements

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// ErrNoNativeRequirement reports a document that carries no requirement
// heading. The caller decides whether to fall back to the foreign adopter.
var ErrNoNativeRequirement = errors.New("the document holds no requirement heading")

// nativeHeading matches a requirement heading in the docs/changes/ format.
// The separator is an em dash or a hyphen, because real documents use both.
// The change identifier is inserted at parse time, so a heading that belongs
// to another change never matches.
const nativeHeadingPattern = `^###\s+(%s-R\d+)\s+(?:—|-)\s*(.*)$`

// ParseNative extracts the requirement set from a docs/changes/ document. No
// model takes part, so the provenance is observed.
func ParseNative(changeID, sourcePath string, body []byte) (Set, error) {
	changeID = strings.TrimSpace(changeID)
	if changeID == "" {
		return Set{}, errors.New("change_id is required")
	}

	pattern, err := regexp.Compile(fmt.Sprintf(nativeHeadingPattern, regexp.QuoteMeta(changeID)))
	if err != nil {
		return Set{}, fmt.Errorf("compile heading pattern: %w", err)
	}

	set := Set{
		SchemaVersion: SchemaVersion,
		ChangeID:      changeID,
		Adopter:       AdopterNative,
		SourcePath:    sourcePath,
	}

	seen := map[string]struct{}{}
	inFence := false
	scanner := bufio.NewScanner(bytes.NewReader(body))
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()

		// A fenced block holds example text. A heading inside one is not a
		// requirement, and a document that documents its own format would
		// otherwise adopt its own example.
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}

		match := pattern.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		id := match[1]
		statement := strings.TrimSpace(match[2])
		if statement == "" {
			return Set{}, fmt.Errorf("requirement heading %q carries no title", id)
		}
		if _, exists := seen[id]; exists {
			return Set{}, fmt.Errorf("duplicate requirement id %q", id)
		}
		seen[id] = struct{}{}
		set.Requirements = append(set.Requirements, Requirement{
			ID:           id,
			Statement:    statement,
			SourceAnchor: strings.TrimSpace(line),
			Provenance:   Observed,
		})
	}
	if err := scanner.Err(); err != nil {
		return Set{}, fmt.Errorf("read document: %w", err)
	}
	if len(set.Requirements) == 0 {
		return Set{}, ErrNoNativeRequirement
	}
	if err := set.Validate(); err != nil {
		return Set{}, err
	}
	return set, nil
}

// AdoptNative reads a document and parses it with the native adopter.
func AdoptNative(changeID, sourcePath string) (Set, error) {
	body, err := os.ReadFile(sourcePath)
	if err != nil {
		return Set{}, fmt.Errorf("read source document: %w", err)
	}
	return ParseNative(changeID, sourcePath, body)
}
