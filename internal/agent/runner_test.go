package agent

import (
	"context"
	"reflect"
	"testing"
)

// stubRunner implements AgentRunner and nothing more.
type stubRunner struct {
	status RunnerStatus
	result RunResult
}

func (runner stubRunner) Probe(ctx context.Context) (RunnerStatus, error) {
	return runner.status, nil
}

func (runner stubRunner) Run(ctx context.Context, req RunRequest) (RunResult, error) {
	return runner.result, nil
}

// P1: the compile-time assertion proves the interface is satisfiable by a type
// that defines only Probe and Run.
var _ AgentRunner = stubRunner{}

func TestAgentRunnerExposesProbeAndRunOnly(t *testing.T) {
	runnerType := reflect.TypeOf((*AgentRunner)(nil)).Elem()
	if runnerType.NumMethod() != 2 {
		t.Fatalf("AgentRunner must expose 2 methods, got %d", runnerType.NumMethod())
	}
	want := map[string]bool{"Probe": true, "Run": true}
	for index := 0; index < runnerType.NumMethod(); index++ {
		name := runnerType.Method(index).Name
		if !want[name] {
			t.Fatalf("unexpected method %q on AgentRunner", name)
		}
	}
}

func TestRunnerStatusUsable(t *testing.T) {
	cases := []struct {
		name   string
		status RunnerStatus
		want   bool
	}{
		{"installed and authenticated", RunnerStatus{Installed: true, Authenticated: true}, true},
		{"not authenticated", RunnerStatus{Installed: true}, false},
		{"not installed", RunnerStatus{Authenticated: true}, false},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := testCase.status.Usable(); got != testCase.want {
				t.Fatalf("Usable() = %t, want %t", got, testCase.want)
			}
		})
	}
}
