package language

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Glossary struct {
	TechnicalNames []string `json:"technical_names"`
	TechnicalVerbs []string `json:"technical_verbs"`
}

func LoadGlossary(path string) (Glossary, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Glossary{}, nil
		}
		return Glossary{}, fmt.Errorf("open glossary: %w", err)
	}
	defer file.Close()

	var glossary Glossary
	section := ""
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		switch line {
		case "technical_names:":
			section = "names"
			continue
		case "technical_verbs:":
			section = "verbs"
			continue
		}
		if strings.HasPrefix(line, "- ") {
			value := strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, "- ")), `"'`)
			switch section {
			case "names":
				glossary.TechnicalNames = append(glossary.TechnicalNames, value)
			case "verbs":
				glossary.TechnicalVerbs = append(glossary.TechnicalVerbs, value)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return Glossary{}, fmt.Errorf("read glossary: %w", err)
	}
	return glossary, nil
}
