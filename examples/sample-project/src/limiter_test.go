package src

import "testing"

// TestRejectsOverTheLimit is the proof for SP-1-R1.
func TestRejectsOverTheLimit(t *testing.T) {
	if !Allow(4, 5) {
		t.Fatal("a caller under the limit must be allowed")
	}
	if Allow(5, 5) {
		t.Fatal("a caller at the limit must be rejected")
	}
	if Allow(6, 5) {
		t.Fatal("a caller over the limit must be rejected")
	}
}
