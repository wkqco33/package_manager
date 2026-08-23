package ui

import "testing"

func TestSpinnerStopBeforeStartIsSafe(t *testing.T) {
	spinner := NewSpinner("working")
	spinner.Stop()
	spinner.Stop()
	spinner.Start()
	spinner.Stop()
}

func TestSpinnerCanBeStoppedMoreThanOnce(t *testing.T) {
	spinner := NewSpinner("working")
	spinner.Start()
	spinner.Stop()
	spinner.Stop()
}
