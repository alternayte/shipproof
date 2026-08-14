package shaping

import "testing"

func TestSessionValidate(t *testing.T) {
	t.Parallel()

	session := Session{
		SchemaVersion: "0.1",
		Subject:       "Webhook retries",
		DocumentKind:  "prd",
		State:         StateShaping,
	}
	if err := session.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestSessionRejectsInvalidState(t *testing.T) {
	t.Parallel()

	session := Session{
		SchemaVersion: "0.1",
		Subject:       "Webhook retries",
		DocumentKind:  "prd",
		State:         "perfect",
	}
	if err := session.Validate(); err == nil {
		t.Fatal("Validate() expected error")
	}
}
