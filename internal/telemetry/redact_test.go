package telemetry

import (
	"strings"
	"testing"
)

func TestRedactMasksKnownSecretShapes(t *testing.T) {
	t.Parallel()

	input := `token: "ghp_abcdefghijklmnopqrstuvwxyz012345" api key: lin_api_1234567890abcdefghijkl password: "hunter2secret" bearer abcdefghijklmnopqrstuvwxyz0123
	sk-proj-abcdefghijklmnopqrstuvwxyz123
	AKIAIOSFODNN7EXAMPLE`

	output := string(Redact([]byte(input)))

	for _, leak := range []string{
		"ghp_abcdefghijklmnopqrstuvwxyz012345",
		"lin_api_1234567890abcdefghijkl",
		"hunter2secret",
		"sk-proj-abcdefghijklmnopqrstuvwxyz123",
		"AKIAIOSFODNN7EXAMPLE",
		"abcdefghijklmnopqrstuvwxyz0123",
	} {
		if strings.Contains(output, leak) {
			t.Errorf("expected %q to be redacted, got: %s", leak, output)
		}
	}

	if !strings.Contains(output, "[REDACTED]") {
		t.Errorf("expected redaction marker in output: %s", output)
	}
}

func TestRedactLeavesOrdinaryTextAlone(t *testing.T) {
	t.Parallel()

	input := "The verification command returned exit code 0. Duration was 1200 ms."
	output := string(Redact([]byte(input)))
	if output != input {
		t.Errorf("expected unchanged text, got %q", output)
	}
}
