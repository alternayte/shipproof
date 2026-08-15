package github

import (
	"encoding/json"
	"fmt"
)

const findPRByCommitQuery = `
query($owner: String!, $name: String!, $sha: String!) {
  repository(owner: $owner, name: $name) {
    object(expression: $sha) {
      ... on Commit {
        associatedPullRequests(first: 1) {
          nodes {
            number
            url
            createdAt
            state
            reviews(last: 100) {
              totalCount
              nodes {
                submittedAt
                author { login }
              }
            }
            reviewThreads(first: 1) { totalCount }
          }
        }
      }
    }
  }
}`

type Review struct {
	Author struct {
		Login string `json:"login"`
	} `json:"author"`
	SubmittedAt string `json:"submittedAt"`
	State       string `json:"state"`
}

type PullRequest struct {
	Number    int    `json:"number"`
	URL       string `json:"url"`
	CreatedAt string `json:"createdAt"`
	State     string `json:"state"`
	Reviews   struct {
		TotalCount int      `json:"totalCount"`
		Nodes      []Review `json:"nodes"`
	} `json:"reviews"`
	ReviewThreads struct {
		TotalCount int `json:"totalCount"`
	} `json:"reviewThreads"`
}

type prQueryData struct {
	Repository struct {
		Object struct {
			AssociatedPullRequests struct {
				Nodes []PullRequest `json:"nodes"`
			} `json:"associatedPullRequests"`
		} `json:"object"`
	} `json:"repository"`
}

type prQueryResponse struct {
	Data prQueryData `json:"data"`
}

func (c *Client) FindPRByCommit(owner, name, sha string) (*PullRequest, error) {
	resp, err := c.execute(findPRByCommitQuery, map[string]any{
		"owner": owner,
		"name":  name,
		"sha":   sha,
	})
	if err != nil {
		return nil, err
	}

	var data prQueryData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, fmt.Errorf("parse PR response: %w", err)
	}

	nodes := data.Repository.Object.AssociatedPullRequests.Nodes
	if len(nodes) == 0 {
		return nil, ErrNotFound
	}

	pr := nodes[0]
	return &pr, nil
}
