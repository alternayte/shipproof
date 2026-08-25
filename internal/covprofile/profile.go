// Package covprofile parses a Go coverage profile and answers one question:
// what does the profile say about one line of one file.
//
// A profile lists only instrumented blocks. A line outside every block never
// appears, and the profile makes no claim about it. That third state is the
// reason this package exists. A parser that returned a boolean would report a
// declaration line as unproven code.
package covprofile

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
)

// State names what a profile says about one line.
type State string

const (
	// Executed holds when the line sits inside a block with a count above zero.
	Executed State = "executed"
	// NotExecuted holds when the line sits inside a block with a zero count.
	NotExecuted State = "not-executed"
	// NotInstrumented holds when no block contains the line. No claim follows.
	NotInstrumented State = "not-instrumented"
)

// Block is one instrumented region of one file.
type Block struct {
	StartLine   int
	StartColumn int
	EndLine     int
	EndColumn   int
	Statements  int
	Count       int
}

// Profile holds every block, keyed by repository-relative file path.
type Profile struct {
	blocks map[string][]Block
}

// ModulePath reads the module path from go.mod. It returns an empty string
// when no module path is available, and the parser then treats every profile
// path as repository-relative already.
func ModulePath(root string) string {
	data, err := os.ReadFile(path.Join(root, "go.mod"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if rest, ok := strings.CutPrefix(trimmed, "module "); ok {
			return strings.TrimSpace(rest)
		}
	}
	return ""
}

// ParseFile reads one profile from disk.
func ParseFile(profilePath, modulePath string) (*Profile, error) {
	file, err := os.Open(profilePath)
	if err != nil {
		return nil, fmt.Errorf("open coverage profile: %w", err)
	}
	defer file.Close()
	return Parse(file, modulePath)
}

// Parse reads one profile. modulePath is stripped from each recorded path, so
// that every key is repository-relative.
func Parse(reader io.Reader, modulePath string) (*Profile, error) {
	profile := &Profile{blocks: map[string][]Block{}}
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "mode:") {
			continue
		}
		file, block, err := parseBlockLine(line)
		if err != nil {
			return nil, err
		}
		profile.blocks[relative(file, modulePath)] = append(profile.blocks[relative(file, modulePath)], block)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read coverage profile: %w", err)
	}
	return profile, nil
}

// parseBlockLine reads one line of the form
// path/file.go:12.34,15.2 3 1.
func parseBlockLine(line string) (string, Block, error) {
	fields := strings.Fields(line)
	if len(fields) != 3 {
		return "", Block{}, fmt.Errorf("coverage profile line %q has %d fields; 3 are required", line, len(fields))
	}
	separator := strings.LastIndex(fields[0], ":")
	if separator < 0 {
		return "", Block{}, fmt.Errorf("coverage profile line %q names no file", line)
	}
	file := fields[0][:separator]
	positions := strings.Split(fields[0][separator+1:], ",")
	if len(positions) != 2 {
		return "", Block{}, fmt.Errorf("coverage profile line %q has no start and end position", line)
	}
	startLine, startColumn, err := parsePosition(positions[0])
	if err != nil {
		return "", Block{}, fmt.Errorf("coverage profile line %q: %w", line, err)
	}
	endLine, endColumn, err := parsePosition(positions[1])
	if err != nil {
		return "", Block{}, fmt.Errorf("coverage profile line %q: %w", line, err)
	}
	statements, err := strconv.Atoi(fields[1])
	if err != nil {
		return "", Block{}, fmt.Errorf("coverage profile line %q has no statement count", line)
	}
	count, err := strconv.Atoi(fields[2])
	if err != nil {
		return "", Block{}, fmt.Errorf("coverage profile line %q has no execution count", line)
	}
	return file, Block{
		StartLine:   startLine,
		StartColumn: startColumn,
		EndLine:     endLine,
		EndColumn:   endColumn,
		Statements:  statements,
		Count:       count,
	}, nil
}

func parsePosition(text string) (int, int, error) {
	parts := strings.Split(text, ".")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("position %q is not line.column", text)
	}
	line, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("position %q has no line", text)
	}
	column, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("position %q has no column", text)
	}
	return line, column, nil
}

// relative strips the module path prefix from a recorded profile path.
func relative(file, modulePath string) string {
	if modulePath == "" {
		return file
	}
	if rest, ok := strings.CutPrefix(file, modulePath+"/"); ok {
		return rest
	}
	return file
}

// Merge adds every block of other to this profile. A nil other is a no-op.
func (profile *Profile) Merge(other *Profile) {
	if other == nil {
		return
	}
	for file, blocks := range other.blocks {
		profile.blocks[file] = append(profile.blocks[file], blocks...)
	}
}

// Lookup answers the three-state question for one line.
//
// Executed wins over NotExecuted. Two blocks can overlap, and one proof that
// ran the line is enough to say a proof ran it.
func (profile *Profile) Lookup(file string, line int) State {
	blocks, known := profile.blocks[file]
	if !known {
		return NotInstrumented
	}
	state := NotInstrumented
	for _, block := range blocks {
		if line < block.StartLine || line > block.EndLine {
			continue
		}
		if block.Count > 0 {
			return Executed
		}
		state = NotExecuted
	}
	return state
}

// Covers reports whether the profile holds any block for this file.
func (profile *Profile) Covers(file string) bool {
	return len(profile.blocks[file]) > 0
}

// Files lists every file the profile names, in sorted order.
func (profile *Profile) Files() []string {
	files := make([]string, 0, len(profile.blocks))
	for file := range profile.blocks {
		files = append(files, file)
	}
	sort.Strings(files)
	return files
}
