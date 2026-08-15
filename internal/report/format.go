package report

import (
	"fmt"
	"strings"
)

func formatPercent(v float64) string {
	if v == 0 {
		return "—"
	}
	return fmt.Sprintf("%.1f%%", v)
}

func formatInt64(n int64) string {
	if n == 0 {
		return "—"
	}
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1000000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000.0)
	}
	return fmt.Sprintf("%.1fM", float64(n)/1000000.0)
}

func formatModelList(models []string) string {
	if len(models) == 0 {
		return "—"
	}
	return strings.Join(models, ", ")
}

func formatDecimal(v float64) string {
	if v == 0 {
		return "—"
	}
	return fmt.Sprintf("%.1f", v)
}
