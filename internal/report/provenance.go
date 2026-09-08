package report

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/alternayte/shipproof/internal/grade"
	"github.com/alternayte/shipproof/internal/schema"
)

func provenanceBadge(kind schema.ProvenanceKind) template.HTML {
	value := grade.FromProvenance(kind)
	return template.HTML(`<span class="prov-badge prov-` + string(value) + `">` +
		string(value) + `</span>`)
}

// gradeBadge renders a grade that a pack already recorded. Section 6 fixes the
// three values, and an unrecognised value renders as claimed, because an
// unknown label is not evidence.
func gradeBadge(value string) template.HTML {
	switch grade.Grade(value) {
	case grade.Observed, grade.Stated:
	default:
		value = string(grade.Claimed)
	}
	return template.HTML(`<span class="prov-badge prov-` + value + `">` + value + `</span>`)
}

func provenanceLabel(kind schema.ProvenanceKind) string {
	return "[" + string(grade.FromProvenance(kind)) + "]"
}

func statusClass(status string) string {
	switch status {
	case "pass":
		return "status-pass"
	case "fail":
		return "status-fail"
	case "skip":
		return "status-skip"
	case "unknown":
		return "status-unknown"
	default:
		return ""
	}
}

func statusIcon(status string) template.HTML {
	switch status {
	case "pass":
		return "&#10004;"
	case "fail":
		return "&#10008;"
	case "skip":
		return "&#10141;"
	case "unknown":
		return "?"
	default:
		return template.HTML(htmlEscape(status))
	}
}

func statusLabel(status string) string {
	switch status {
	case "pass":
		return "PASS"
	case "fail":
		return "FAIL"
	case "skip":
		return "SKIP"
	case "unknown":
		return "UNKNOWN"
	default:
		return strings.ToUpper(status)
	}
}

func htmlEscape(s string) string {
	var sb strings.Builder
	for _, r := range s {
		switch r {
		case '&':
			sb.WriteString("&amp;")
		case '<':
			sb.WriteString("&lt;")
		case '>':
			sb.WriteString("&gt;")
		case '"':
			sb.WriteString("&#34;")
		case '\'':
			sb.WriteString("&#39;")
		default:
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func formatTokens(tokens int64) string {
	if tokens == 0 {
		return "—"
	}
	if tokens < 1000 {
		return fmt.Sprintf("%d", tokens)
	}
	return fmt.Sprintf("%.1fk", float64(tokens)/1000.0)
}

func formatDollars(cost float64) string {
	if cost == 0 {
		return "—"
	}
	return fmt.Sprintf("$%.2f", cost)
}

func formatDuration(startedAt, endedAt string) string {
	if startedAt == "" || endedAt == "" {
		return "—"
	}
	return startedAt + " → " + endedAt
}
