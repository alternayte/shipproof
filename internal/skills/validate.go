package skills

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var namePattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type Metadata struct {
	Name        string
	Description string
}

func ValidateDirectory(path string) error {
	skillPath := filepath.Join(path, "SKILL.md")
	metadata, lines, err := parseSkill(skillPath)
	if err != nil {
		return err
	}

	if metadata.Name == "" {
		return errors.New("name is required")
	}
	if len(metadata.Name) > 64 || !namePattern.MatchString(metadata.Name) {
		return fmt.Errorf("name %q does not match the Agent Skills naming rules", metadata.Name)
	}
	if filepath.Base(filepath.Clean(path)) != metadata.Name {
		return fmt.Errorf("name %q must match parent directory %q", metadata.Name, filepath.Base(filepath.Clean(path)))
	}
	if metadata.Description == "" {
		return errors.New("description is required")
	}
	if len(metadata.Description) > 1024 {
		return errors.New("description exceeds 1024 characters")
	}
	if lines > 500 {
		return fmt.Errorf("SKILL.md has %d lines; keep it under 500 and move detail to references", lines)
	}
	return nil
}

func ValidateCatalog(root string) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		return fmt.Errorf("read skills catalog: %w", err)
	}
	var problems []string
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "evals" {
			continue
		}
		path := filepath.Join(root, entry.Name())
		if _, err := os.Stat(filepath.Join(path, "SKILL.md")); err != nil {
			continue
		}
		if err := ValidateDirectory(path); err != nil {
			problems = append(problems, entry.Name()+": "+err.Error())
		}
	}
	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "\n"))
	}
	return nil
}

func parseSkill(path string) (Metadata, int, error) {
	file, err := os.Open(path)
	if err != nil {
		return Metadata{}, 0, fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineCount := 0
	inFrontmatter := false
	frontmatterClosed := false
	metadata := Metadata{}

	for scanner.Scan() {
		lineCount++
		line := scanner.Text()
		if lineCount == 1 {
			if strings.TrimSpace(line) != "---" {
				return Metadata{}, lineCount, errors.New("SKILL.md must start with YAML frontmatter")
			}
			inFrontmatter = true
			continue
		}
		if inFrontmatter && strings.TrimSpace(line) == "---" {
			inFrontmatter = false
			frontmatterClosed = true
			continue
		}
		if inFrontmatter {
			key, value, ok := strings.Cut(line, ":")
			if !ok {
				continue
			}
			value = strings.Trim(strings.TrimSpace(value), `"'`)
			switch strings.TrimSpace(key) {
			case "name":
				metadata.Name = value
			case "description":
				metadata.Description = value
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return Metadata{}, lineCount, fmt.Errorf("read %s: %w", path, err)
	}
	if !frontmatterClosed {
		return Metadata{}, lineCount, errors.New("YAML frontmatter is not closed")
	}
	return metadata, lineCount, nil
}
