package linear

import "encoding/json"

type Issue struct {
	ID          string     `json:"id"`
	Identifier  string     `json:"identifier"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	State       string     `json:"state"`
	Assignee    string     `json:"assignee,omitempty"`
	CycleNumber int        `json:"cycle_number,omitempty"`
	Labels      []string   `json:"labels,omitempty"`
	Documents   []Document `json:"documents,omitempty"`
	URL         string     `json:"url"`
}

type Document struct {
	Title   string `json:"title"`
	Content string `json:"content,omitempty"`
	URL     string `json:"url,omitempty"`
}

type Project struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Description string  `json:"description,omitempty"`
	State       string  `json:"state"`
	Lead        string  `json:"lead,omitempty"`
	Issues      []Issue `json:"issues,omitempty"`
	URL         string  `json:"url"`
}

type PlanIssue struct {
	Title       string      `json:"title"`
	Description string      `json:"description,omitempty"`
	Children    []PlanIssue `json:"children,omitempty"`
}

type SyncResult struct {
	ProjectID     string         `json:"project_id"`
	ProjectURL    string         `json:"project_url"`
	CreatedIssues []CreatedIssue `json:"created_issues"`
}

type CreatedIssue struct {
	Title      string `json:"title"`
	Identifier string `json:"identifier"`
	URL        string `json:"url"`
}

type graphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type graphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []graphQLError  `json:"errors,omitempty"`
}

type graphQLError struct {
	Message string `json:"message"`
}
