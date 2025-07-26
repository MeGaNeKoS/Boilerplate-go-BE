package enums

import "testing"

func TestDirtyStrategyValues(t *testing.T) {
	if string(DirtyStrategyRetry) != "retry" {
		t.Fatalf("unexpected retry string")
	}
	if string(DirtyStrategySkip) != "skip" {
		t.Fatalf("unexpected skip string")
	}
}
