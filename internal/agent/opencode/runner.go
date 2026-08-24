package opencode

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/alternayte/shipproof/internal/agent"
)

// Name is the registry name of this runner.
const Name = "opencode"

const defaultBinary = "opencode"

// Runner executes work through an OpenCode server. When `base_url` is set it
// uses that server. Otherwise it manages a local server process for the life
// of one call.
type Runner struct {
	baseURL string
	binary  string
}

// New builds an OpenCode runner from its configuration. The configuration must
// not hold a credential. OpenCode owns provider authentication.
func New(config agent.RunnerConfig) (agent.AgentRunner, error) {
	binary := strings.TrimSpace(config.Setting("path"))
	if binary == "" {
		binary = defaultBinary
	}
	return Runner{baseURL: strings.TrimSpace(config.Setting("base_url")), binary: binary}, nil
}

func capabilities() agent.RunnerCapabilities {
	return agent.RunnerCapabilities{
		Resume:           true,
		ReadOnly:         false,
		WorkspaceWrite:   true,
		StructuredOutput: true,
		Streaming:        false,
	}
}

// Probe reports server availability and provider status. It never reports
// credential content.
func (runner Runner) Probe(ctx context.Context) (agent.RunnerStatus, error) {
	status := agent.RunnerStatus{Capabilities: capabilities()}

	// Probe never starts a server process. A probe must stay fast and free of
	// side effects. Set `agent.runners.opencode.base_url` to a running server.
	if runner.baseURL == "" {
		status.Detail = "OpenCode server address not set. Start `opencode serve` and set agent.runners.opencode.base_url."
		return status, nil
	}
	server := newClient(runner.baseURL)

	info, err := server.app(ctx)
	if err != nil {
		status.Detail = "OpenCode server did not answer /app."
		return status, nil
	}
	status.Installed = true
	status.Version = info.Version

	providers, err := server.providers(ctx)
	if err != nil || len(providers.Providers) == 0 {
		status.Detail = "OpenCode has no connected provider. Run `opencode auth login`."
		return status, nil
	}
	status.Authenticated = true
	status.Detail = "OpenCode server is ready."
	return status, nil
}

// Run executes one bounded coding task over the server transport. The reported
// status is a runner claim. It is never evidence.
func (runner Runner) Run(ctx context.Context, req agent.RunRequest) (agent.RunResult, error) {
	server, stop, err := runner.server(ctx, req.Workspace)
	if err != nil {
		return agent.RunResult{}, err
	}
	defer stop()

	title := req.Change.ID
	if title == "" {
		title = "shipproof"
	}
	session, err := server.createSession(ctx, title)
	if err != nil {
		return agent.RunResult{}, fmt.Errorf("create opencode session: %w", err)
	}

	message, err := server.sendMessage(ctx, session.ID, agent.BuildPrompt(req))
	if err != nil {
		return agent.RunResult{
			Status:     agent.RunStatusFailed,
			Summary:    err.Error(),
			SessionRef: session.ID,
			Metadata:   map[string]string{"transport": "server"},
		}, nil
	}

	return agent.RunResult{
		Status:     agent.RunStatusSuccess,
		Summary:    message.text(),
		SessionRef: session.ID,
		Metadata:   map[string]string{"transport": "server"},
	}, nil
}

// server returns a client and a stop function. A configured base URL needs no
// process. Otherwise the runner starts a managed local server.
func (runner Runner) server(ctx context.Context, workspace string) (*client, func(), error) {
	if runner.baseURL != "" {
		return newClient(runner.baseURL), func() {}, nil
	}

	binary, err := exec.LookPath(runner.binary)
	if err != nil {
		return nil, nil, fmt.Errorf("opencode CLI not found: %w", err)
	}
	port, err := freePort()
	if err != nil {
		return nil, nil, err
	}

	cmd := exec.Command(binary, "serve", "--port", strconv.Itoa(port))
	cmd.Dir = workspace
	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("start opencode server: %w", err)
	}
	stop := func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}

	server := newClient(fmt.Sprintf("http://127.0.0.1:%d", port))
	if err := waitReady(ctx, server); err != nil {
		stop()
		return nil, nil, err
	}
	return server, stop, nil
}

func waitReady(ctx context.Context, server *client) error {
	deadline := time.Now().Add(20 * time.Second)
	for {
		if _, err := server.app(ctx); err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("opencode server did not become ready")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
}

func freePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("reserve port: %w", err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}
