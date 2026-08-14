package linear

import (
	"encoding/json"
	"fmt"
	"strings"
)

const projectQuery = `
query GetProject($slug: String!) {
  project(slugId: $slug) {
    id
    name
    slugId
    description
    state
    lead { name }
    url
    issues {
      nodes {
        id
        identifier
        title
        state { name }
        url
      }
    }
  }
}`

type projectAPIResponse struct {
	Project *projectAPIData `json:"project"`
}

type projectAPIData struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	SlugID      string `json:"slugId"`
	Description string `json:"description"`
	State       string `json:"state"`
	Lead        *struct {
		Name string `json:"name"`
	} `json:"lead"`
	URL    string `json:"url"`
	Issues struct {
		Nodes []projectIssueData `json:"nodes"`
	} `json:"issues"`
}

type projectIssueData struct {
	ID         string `json:"id"`
	Identifier string `json:"identifier"`
	Title      string `json:"title"`
	State      struct {
		Name string `json:"name"`
	} `json:"state"`
	URL string `json:"url"`
}

func GetProject(client *Client, slug string) (Project, error) {
	if strings.TrimSpace(slug) == "" {
		return Project{}, fmt.Errorf("project name or slug is required")
	}

	resp, err := client.execute(projectQuery, map[string]any{"slug": slug})
	if err != nil {
		return Project{}, err
	}

	var apiResp projectAPIResponse
	if err := json.Unmarshal(resp.Data, &apiResp); err != nil {
		return Project{}, fmt.Errorf("parse project data: %w", err)
	}

	if apiResp.Project == nil {
		return Project{}, fmt.Errorf("%w: project %q", ErrNotFound, slug)
	}

	project := Project{
		ID:          apiResp.Project.ID,
		Name:        apiResp.Project.Name,
		Slug:        apiResp.Project.SlugID,
		Description: apiResp.Project.Description,
		State:       apiResp.Project.State,
		URL:         apiResp.Project.URL,
	}

	if apiResp.Project.Lead != nil {
		project.Lead = apiResp.Project.Lead.Name
	}

	for _, i := range apiResp.Project.Issues.Nodes {
		project.Issues = append(project.Issues, Issue{
			ID:         i.ID,
			Identifier: i.Identifier,
			Title:      i.Title,
			State:      i.State.Name,
			URL:        i.URL,
		})
	}

	return project, nil
}
