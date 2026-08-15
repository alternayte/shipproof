package report

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/shipproof/shipproof/internal/schema"
)

func provenanceBadge(kind schema.ProvenanceKind) template.HTML {
	switch kind {
	case schema.ProvenanceObserved:
		return `<span class="prov-badge prov-observed">observed</span>`
	case schema.ProvenanceDerived:
		return `<span class="prov-badge prov-derived">derived</span>`
	case schema.ProvenanceInferred:
		return `<span class="prov-badge prov-inferred">inferred</span>`
	case schema.ProvenanceHuman:
		return `<span class="prov-badge prov-human">human</span>`
	default:
		return template.HTML(`<span class="prov-badge">` + htmlEscape(string(kind)) + `</span>`)
	}
}

func provenanceLabel(kind schema.ProvenanceKind) string {
	return "[" + string(kind) + "]"
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
