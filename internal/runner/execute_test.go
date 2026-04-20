package runner

import (
	"testing"

	"devsetup/internal/types"
)

func TestExitCode(t *testing.T) {
	if got := ExitCode(types.RunSummary{Failed: 0}); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
	if got := ExitCode(types.RunSummary{Failed: 1}); got != 1 {
		t.Fatalf("expected 1, got %d", got)
	}
}
