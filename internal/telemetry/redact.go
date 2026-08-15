package telemetry

import "regexp"

const redactionMarker = "[REDACTED]"

// secretPatterns matches common credential shapes in transcripts. Redaction is
// best-effort and deterministic. It is not a guarantee against every secret
// form; full capture stores material verbatim.
var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bsk-[A-Za-z0-9_-]{12,}\b`),
	regexp.MustCompile(`\bgh[pousr]_[A-Za-z0-9]{20,}\b`),
	regexp.MustCompile(`\bgithub_pat_[A-Za-z0-9_]{20,}\b`),
	regexp.MustCompile(`\blin_api_[A-Za-z0-9]{20,}\b`),
	regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`),
	regexp.MustCompile(`(?i)\bxox[baprs]-[A-Za-z0-9-]{10,}\b`),
	regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9._~+/=-]{8,}`),
	regexp.MustCompile(`(?i)\b(api[_-]?key|apikey|secret|token|password|passwd)\b\s*["']?\s*[:=]\s*["'][^"']{4,}["']`),
}

// Redact replaces recognized secret shapes with a marker.
func Redact(content []byte) []byte {
	result := content
	for _, pattern := range secretPatterns {
		result = pattern.ReplaceAll(result, []byte(redactionMarker))
	}
	return result
}
