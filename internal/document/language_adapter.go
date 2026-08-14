package document

import "github.com/shipproof/shipproof/internal/language"

func languageFindings(content string, glossary language.Glossary, options language.Options) []Finding {
	items := language.Lint(content, glossary, options)
	findings := make([]Finding, 0, len(items))
	for _, item := range items {
		findings = append(findings, Finding{
			ID:              item.ID,
			Class:           FindingClass(item.Class),
			Category:        item.Category,
			Line:            item.Line,
			Claim:           item.Claim,
			Reason:          item.Reason,
			SuggestedAction: item.SuggestedAction,
			Source:          item.Source,
		})
	}
	return findings
}
