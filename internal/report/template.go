package report

import (
	"embed"
	"html/template"
	"io"
)

//go:embed templates/*
var templateFS embed.FS

var templates = template.Must(template.New("").Funcs(template.FuncMap{
	"badge":           provenanceBadge,
	"statusClass":     statusClass,
	"statusIcon":      statusIcon,
	"statusLabel":     statusLabel,
	"formatTokens":    formatTokens,
	"formatDollars":   formatDollars,
	"formatDuration":  formatDuration,
	"formatPercent":   formatPercent,
	"formatInt64":     formatInt64,
	"formatModelList": formatModelList,
	"provLabel":       func() string { return "observed" },
}).ParseFS(templateFS, "templates/*"))

func executeTemplate(w io.Writer, name string, data any) error {
	return templates.ExecuteTemplate(w, name, data)
}
