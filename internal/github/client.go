package github

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const githubGraphQLURL = "https://api.github.com/graphql"

type Client struct {
	token      string
	baseURL    string
	httpClient *http.Client
}

func NewClient(token string) (*Client, error) {
	if strings.TrimSpace(token) == "" {
		return nil, ErrNoToken
	}
	return &Client{
		token:      token,
		baseURL:    githubGraphQLURL,
		httpClient: &http.Client{},
	}, nil
}

func NewClientWithURL(token, baseURL string) (*Client, error) {
	if strings.TrimSpace(token) == "" {
		return nil, ErrNoToken
	}
	return &Client{
		token:      token,
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}, nil
}

type graphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type graphQLResponse struct {
	Data   json.RawMessage   `json:"data"`
	Errors []graphQLError    `json:"errors"`
	Error  *graphQLErrorBody `json:"error,omitempty"`
}

type graphQLError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

type graphQLErrorBody struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

func (c *Client) execute(query string, variables map[string]any) (*graphQLResponse, error) {
	reqBody := graphQLRequest{
		Query:     query,
		Variables: variables,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		msg := strings.TrimSpace(string(respBody))
		if msg == "" {
			msg = http.StatusText(resp.StatusCode)
		}
		return nil, &APIError{StatusCode: resp.StatusCode, Message: msg}
	}

	var gqlResp graphQLResponse
	if err := json.Unmarshal(respBody, &gqlResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if len(gqlResp.Errors) > 0 {
		msgs := make([]string, len(gqlResp.Errors))
		notFound := false
		for i, e := range gqlResp.Errors {
			msgs[i] = e.Message
			if strings.EqualFold(e.Type, "NOT_FOUND") {
				notFound = true
			}
		}
		combined := strings.Join(msgs, "; ")
		lower := strings.ToLower(combined)
		if notFound || strings.Contains(lower, "not found") || strings.Contains(lower, "could not resolve") {
			return nil, ErrNotFound
		}
		return nil, errors.New(combined)
	}

	return &gqlResp, nil
}
