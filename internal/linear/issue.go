package linear

import (
	"encoding/json"
	"fmt"
	"regexp"
)

var issueIDPattern = regexp.MustCompile(`^[A-Z]{2,8}-\d+$`)

const issueQuery = `
query GetIssue($id: String!) {
  issue(id: $id) {
    id
    identifier
    title
    description
    state { name }
    assignee { name }
    cycle { number }
    labels { nodes { name } }
    url
    attachments {
      nodes {
        title
        url
      }
    }
  }
}`

type issueAPIResponse struct {
	Issue *issueAPIData `json:"issue"`
}

type issueAPIData struct {
	ID          string `json:"id"`
	Identifier  string `json:"identifier"`
	Title       string `json:"title"`
	Description string `json:"description"`
	State       struct {
		Name string `json:"name"`
	} `json:"state"`
	Assignee *struct {
		Name string `json:"name"`
	} `json:"assignee"`
	Cycle *struct {
		Number float64 `json:"number"`
	} `json:"cycle"`
	Labels struct {
		Nodes []struct {
			Name string `json:"name"`
		} `json:"nodes"`
	} `json:"labels"`
	URL         string `json:"url"`
	Attachments struct {
		Nodes []struct {
			Title string `json:"title"`
			URL   string `json:"url"`
		} `json:"nodes"`
	} `json:"attachments"`
}

func GetIssue(client *Client, identifier string) (Issue, error) {
	if !issueIDPattern.MatchString(identifier) {
		return Issue{}, fmt.Errorf("invalid issue identifier %q; use the format TEAM-123", identifier)
	}

	resp, err := client.execute(issueQuery, map[string]any{"id": identifier})
	if err != nil {
		return Issue{}, err
	}

	var apiResp issueAPIResponse
	if err := json.Unmarshal(resp.Data, &apiResp); err != nil {
		return Issue{}, fmt.Errorf("parse issue data: %w", err)
	}

	if apiResp.Issue == nil {
		return Issue{}, fmt.Errorf("%w: issue %q", ErrNotFound, identifier)
	}

	issue := Issue{
		ID:          apiResp.Issue.ID,
		Identifier:  apiResp.Issue.Identifier,
		Title:       apiResp.Issue.Title,
		Description: apiResp.Issue.Description,
		State:       apiResp.Issue.State.Name,
		URL:         apiResp.Issue.URL,
	}

	if apiResp.Issue.Assignee != nil {
		issue.Assignee = apiResp.Issue.Assignee.Name
	}
	if apiResp.Issue.Cycle != nil {
		issue.CycleNumber = int(apiResp.Issue.Cycle.Number)
	}

	for _, l := range apiResp.Issue.Labels.Nodes {
		issue.Labels = append(issue.Labels, l.Name)
	}

	for _, a := range apiResp.Issue.Attachments.Nodes {
		issue.Documents = append(issue.Documents, Document{
			Title: a.Title,
			URL:   a.URL,
		})
	}

	return issue, nil
}
