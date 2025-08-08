package helpers

import "testing"

func TestInternalTag(t *testing.T) {
	if got := InternalTag(); got != hideFromPublicTag {
		t.Fatalf("tag %q", got)
	}
}
