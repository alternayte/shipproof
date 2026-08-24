package agent

import (
	"strings"
	"testing"
)

func TestPolicyForRoles(t *testing.T) {
	implementer, err := PolicyFor(RoleImplementer)
	if err != nil {
		t.Fatalf("implementer policy: %v", err)
	}
	if !implementer.WorkspaceWrite || implementer.GitPush {
		t.Fatalf("implementer policy = %+v", implementer)
	}

	reviewer, err := PolicyFor(RoleReviewer)
	if err != nil {
		t.Fatalf("reviewer policy: %v", err)
	}
	if reviewer.WorkspaceWrite || reviewer.GitPush {
		t.Fatalf("reviewer policy = %+v", reviewer)
	}

	if _, err := PolicyFor(AgentRole("auditor")); err == nil {
		t.Fatal("unknown role must fail")
	}
}

func TestPolicyEnforceable(t *testing.T) {
	reviewer, _ := PolicyFor(RoleReviewer)
	if ok, _ := reviewer.Enforceable(RunnerCapabilities{ReadOnly: true}); !ok {
		t.Fatal("read-only runner must satisfy the reviewer policy")
	}
	ok, detail := reviewer.Enforceable(RunnerCapabilities{WorkspaceWrite: true})
	if ok {
		t.Fatal("a runner without read-only capability must not satisfy the reviewer policy")
	}
	if !strings.Contains(detail, "read-only") {
		t.Fatalf("detail = %q", detail)
	}

	implementer, _ := PolicyFor(RoleImplementer)
	if ok, _ := implementer.Enforceable(RunnerCapabilities{WorkspaceWrite: true}); !ok {
		t.Fatal("workspace-write runner must satisfy the implementer policy")
	}
	if ok, _ := implementer.Enforceable(RunnerCapabilities{ReadOnly: true}); ok {
		t.Fatal("a runner without workspace write must not satisfy the implementer policy")
	}
}

func TestReviewerConstraintsForbidEdits(t *testing.T) {
	reviewer, _ := PolicyFor(RoleReviewer)
	constraints := reviewer.Constraints()
	if len(constraints) == 0 || !strings.Contains(constraints[0], "Do not modify") {
		t.Fatalf("constraints = %v", constraints)
	}
}

func TestBuildPrompt(t *testing.T) {
	prompt := BuildPrompt(RunRequest{
		Change:       Change{ID: "CH-007", Title: "Rotate signing secrets", SnapshotPath: ".shipproof/changes/CH-007/snapshot.md", Intent: "Rotate without downtime."},
		Role:         RoleImplementer,
		Instructions: "Implement the approved change.",
		Constraints:  []string{"Do not push to a remote."},
	})
	for _, want := range []string{"Role: implementer", "Change: CH-007", "Rotate signing secrets", "Implement the approved change.", "Rotate without downtime.", "- Do not push to a remote."} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
}
