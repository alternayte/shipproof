package linear

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

const createProjectMutation = `
mutation CreateProject($name: String!, $teamIds: [String!]!) {
  projectCreate(input: { name: $name, teamIds: $teamIds }) {
    project {
      id
      name
      url
    }
  }
}`

const createIssueMutation = `
mutation CreateIssue($title: String!, $description: String, $projectId: String!, $parentId: String) {
  issueCreate(input: { title: $title, description: $description, projectId: $projectId, parentId: $parentId }) {
    issue {
      id
      identifier
      title
      url
    }
  }
}`

type createProjectResponse struct {
	ProjectCreate struct {
		Project struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"project"`
	} `json:"projectCreate"`
}

type createIssueResponse struct {
	IssueCreate struct {
		Issue struct {
			ID         string `json:"id"`
			Identifier string `json:"identifier"`
			Title      string `json:"title"`
			URL        string `json:"url"`
		} `json:"issue"`
	} `json:"issueCreate"`
}

func SyncPlan(client *Client, planFile string, teamID string, stdin io.Reader, stderr io.Writer) (SyncResult, error) {
	data, err := os.ReadFile(planFile)
	if err != nil {
		return SyncResult{}, fmt.Errorf("read plan file: %w", err)
	}

	var issues []PlanIssue
	if err := json.Unmarshal(data, &issues); err != nil {
		return SyncResult{}, fmt.Errorf("parse plan file: %w", err)
	}

	if len(issues) == 0 {
		return SyncResult{}, fmt.Errorf("plan file contains no issues")
	}

	if !confirmSync(issues, stdin, stderr) {
		return SyncResult{}, fmt.Errorf("sync cancelled")
	}

	projectName := fmt.Sprintf("ShipProof plan — %s", planFile)
	project, err := createProject(client, projectName, teamID)
	if err != nil {
		return SyncResult{}, fmt.Errorf("create project: %w", err)
	}

	result := SyncResult{
		ProjectID:  project.ID,
		ProjectURL: project.URL,
	}

	for _, issue := range issues {
		created, err := createIssueRecursive(client, project.ID, issue, "")
		if err != nil {
			return result, fmt.Errorf("create issue %q: %w", issue.Title, err)
		}
		result.CreatedIssues = append(result.CreatedIssues, created...)
	}

	return result, nil
}

func createIssueRecursive(client *Client, projectID string, plan PlanIssue, parentID string) ([]CreatedIssue, error) {
	issue, err := createIssue(client, plan.Title, plan.Description, projectID, parentID)
	if err != nil {
		return nil, err
	}

	created := []CreatedIssue{issue}

	for _, child := range plan.Children {
		childCreated, err := createIssueRecursive(client, projectID, child, issue.Identifier)
		if err != nil {
			return nil, err
		}
		created = append(created, childCreated...)
	}

	return created, nil
}

func createProject(client *Client, name, teamID string) (*struct{ ID, Name, URL string }, error) {
	if strings.TrimSpace(teamID) == "" {
		return nil, fmt.Errorf("team ID is required for project creation; set LINEAR_TEAM_ID")
	}

	resp, err := client.execute(createProjectMutation, map[string]any{
		"name":    name,
		"teamIds": []string{teamID},
	})
	if err != nil {
		return nil, err
	}

	var apiResp createProjectResponse
	if err := json.Unmarshal(resp.Data, &apiResp); err != nil {
		return nil, fmt.Errorf("parse project create response: %w", err)
	}

	return &struct {
		ID   string
		Name string
		URL  string
	}{
		ID:   apiResp.ProjectCreate.Project.ID,
		Name: apiResp.ProjectCreate.Project.Name,
		URL:  apiResp.ProjectCreate.Project.URL,
	}, nil
}

func createIssue(client *Client, title, description, projectID, parentID string) (CreatedIssue, error) {
	vars := map[string]any{
		"title":       title,
		"description": description,
		"projectId":   projectID,
	}
	if parentID != "" {
		vars["parentId"] = parentID
	}

	resp, err := client.execute(createIssueMutation, vars)
	if err != nil {
		return CreatedIssue{}, err
	}

	var apiResp createIssueResponse
	if err := json.Unmarshal(resp.Data, &apiResp); err != nil {
		return CreatedIssue{}, fmt.Errorf("parse issue create response: %w", err)
	}

	return CreatedIssue{
		Title:      apiResp.IssueCreate.Issue.Title,
		Identifier: apiResp.IssueCreate.Issue.Identifier,
		URL:        apiResp.IssueCreate.Issue.URL,
	}, nil
}

func confirmSync(issues []PlanIssue, stdin io.Reader, stderr io.Writer) bool {
	fmt.Fprintln(stderr, "The following issues will be created in Linear:")
	countIssues(issues, 0, stderr)
	fmt.Fprintln(stderr)
	fmt.Fprint(stderr, "Create these issues? [y/N]: ")

	var response string
	fmt.Fscanln(stdin, &response)
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

func countIssues(issues []PlanIssue, depth int, w io.Writer) {
	indent := strings.Repeat("  ", depth)
	for _, issue := range issues {
		fmt.Fprintf(w, "%s- %s\n", indent, issue.Title)
		countIssues(issue.Children, depth+1, w)
	}
}
