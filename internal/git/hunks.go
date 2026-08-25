package git

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Hunk is one changed region of one file, on the post-image side.
//
// LineCount is zero for a pure deletion. A deleted line is not a line the
// working tree holds, so no proof can run it and no claim is made about it.
type Hunk struct {
	StartLine int
	LineCount int
	// Symbol is the best-effort name from the hunk header. Git derives it from
	// the diff driver, and it can be empty or wrong. The report treats it as a
	// label, never as a fact.
	Symbol string
}

// FileHunks holds every changed region of one file.
type FileHunks struct {
	Path  string
	Hunks []Hunk
}

// CollectChangedHunks returns the changed line ranges between two revisions.
//
// The diff uses no context, so every reported line is a line the change
// touched.
func CollectChangedHunks(dir, base, head string) ([]FileHunks, error) {
	if err := checkRepo(dir); err != nil {
		return nil, err
	}
	rangeString := base + ".." + head
	command := exec.Command("git", "diff", "-U0", "--no-color", rangeString)
	command.Dir = dir

	out, err := command.Output()
	if err != nil {
		if isBadRevision(err) {
			return nil, fmt.Errorf("revision range %q: %w", rangeString, ErrBadRevision)
		}
		return nil, fmt.Errorf("git diff -U0: %w", err)
	}
	return ParseHunks(string(out)), nil
}

// ParseHunks reads a unified diff and returns the post-image ranges.
func ParseHunks(diff string) []FileHunks {
	var files []FileHunks
	// current is an index, not a pointer. An append to files can move the
	// backing array, and a pointer into it would then address the old copy.
	current := -1
	for _, line := range strings.Split(diff, "\n") {
		if strings.HasPrefix(line, "+++ ") {
			path := strings.TrimSpace(strings.TrimPrefix(line, "+++ "))
			if path == "/dev/null" {
				current = -1
				continue
			}
			path = strings.TrimPrefix(path, "b/")
			files = append(files, FileHunks{Path: path})
			current = len(files) - 1
			continue
		}
		if current < 0 || !strings.HasPrefix(line, "@@") {
			continue
		}
		hunk, ok := parseHunkHeader(line)
		if !ok {
			continue
		}
		files[current].Hunks = append(files[current].Hunks, hunk)
	}

	kept := files[:0]
	for _, file := range files {
		if len(file.Hunks) > 0 {
			kept = append(kept, file)
		}
	}
	return kept
}

// parseHunkHeader reads a line of the form
// @@ -12,3 +14,5 @@ func Name().
func parseHunkHeader(line string) (Hunk, bool) {
	rest, ok := strings.CutPrefix(line, "@@ ")
	if !ok {
		return Hunk{}, false
	}
	end := strings.Index(rest, " @@")
	if end < 0 {
		return Hunk{}, false
	}
	ranges := strings.Fields(rest[:end])
	symbol := strings.TrimSpace(rest[end+len(" @@"):])

	for _, item := range ranges {
		text, isNew := strings.CutPrefix(item, "+")
		if !isNew {
			continue
		}
		start, count, ok := parseRange(text)
		if !ok {
			return Hunk{}, false
		}
		return Hunk{StartLine: start, LineCount: count, Symbol: symbol}, true
	}
	return Hunk{}, false
}

// parseRange reads "14,5" or "14". A missing count means one line.
func parseRange(text string) (int, int, bool) {
	start, count := text, "1"
	if comma := strings.Index(text, ","); comma >= 0 {
		start, count = text[:comma], text[comma+1:]
	}
	startLine, err := strconv.Atoi(start)
	if err != nil {
		return 0, 0, false
	}
	lineCount, err := strconv.Atoi(count)
	if err != nil {
		return 0, 0, false
	}
	return startLine, lineCount, true
}
