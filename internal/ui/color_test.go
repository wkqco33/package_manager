package ui

import "testing"

func TestColorCanBeDisabled(t *testing.T) {
	old := ColorEnabled
	t.Cleanup(func() { ColorEnabled = old })
	ColorEnabled = false

	if got := Success("done"); got != "✔ done" {
		t.Fatalf("Success without color = %q, want plain output", got)
	}
	if got := Info("working"); got != "ℹ working" {
		t.Fatalf("Info without color = %q, want plain output", got)
	}
}
