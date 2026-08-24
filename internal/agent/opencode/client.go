// Package opencode adapts an OpenCode server to the ShipProof AgentRunner
// interface. Every HTTP detail stays inside this package.
package opencode

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// client speaks the small subset of the OpenCode server API that ShipProof
// needs: application information, provider status, session creation, and one
// bounded message exchange.
type client struct {
	baseURL string
	http    *http.Client
}

func newClient(baseURL string) *client {
	return &client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 30 * time.Minute},
	}
}

type appInfo struct {
	Version string `json:"version"`
}

type providerList struct {
	Providers []struct {
		ID string `json:"id"`
	} `json:"providers"`
}

type sessionInfo struct {
	ID string `json:"id"`
}

type messagePart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type messageResponse struct {
	Info  sessionInfo   `json:"info"`
	Parts []messagePart `json:"parts"`
}

func (c *client) do(ctx context.Context, method, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		reader = bytes.NewReader(data)
	}

	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := c.http.Do(request)
	if err != nil {
		return fmt.Errorf("call opencode server: %w", err)
	}
	defer response.Body.Close()

	data, err := io.ReadAll(io.LimitReader(response.Body, 8<<20))
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("opencode server returned %s for %s", response.Status, path)
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("parse response from %s: %w", path, err)
	}
	return nil
}

func (c *client) app(ctx context.Context) (appInfo, error) {
	var info appInfo
	err := c.do(ctx, http.MethodGet, "/app", nil, &info)
	return info, err
}

func (c *client) providers(ctx context.Context) (providerList, error) {
	var list providerList
	err := c.do(ctx, http.MethodGet, "/config/providers", nil, &list)
	return list, err
}

func (c *client) createSession(ctx context.Context, title string) (sessionInfo, error) {
	var session sessionInfo
	err := c.do(ctx, http.MethodPost, "/session", map[string]string{"title": title}, &session)
	return session, err
}

func (c *client) sendMessage(ctx context.Context, sessionID, text string) (messageResponse, error) {
	var message messageResponse
	body := map[string]any{
		"parts": []map[string]string{{"type": "text", "text": text}},
	}
	err := c.do(ctx, http.MethodPost, "/session/"+sessionID+"/message", body, &message)
	return message, err
}

func (message messageResponse) text() string {
	var parts []string
	for _, part := range message.Parts {
		if part.Type == "text" && strings.TrimSpace(part.Text) != "" {
			parts = append(parts, strings.TrimSpace(part.Text))
		}
	}
	return strings.Join(parts, "\n")
}
