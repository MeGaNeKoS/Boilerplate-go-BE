package code

import (
	"net/http"
	"testing"
)

func TestCodeMessageHelpers(t *testing.T) {
	c := Code{HTTPCode: http.StatusBadRequest, Message: "bad", InternalCode: 1}
	if got := c.PrependMessage("first"); got.Message != "first: bad" || got.HTTPCode != http.StatusBadRequest || got.InternalCode != 1 {
		t.Fatalf("PrependMessage failed: %#v", got)
	}
	if got := c.AppendMessage("end"); got.Message != "bad: end" {
		t.Fatalf("AppendMessage failed: %#v", got)
	}
	if got := c.ReplaceMessage("new"); got.Message != "new" {
		t.Fatalf("ReplaceMessage failed: %#v", got)
	}
}

func TestCodeError(t *testing.T) {
	c := Code{Message: "boom"}
	if c.Error() != "boom" {
		t.Fatalf("unexpected error %q", c.Error())
	}
}

func TestRegistryAutoName(t *testing.T) {
	if c, ok := Lookup("ErrPayloadError"); !ok || c.HTTPCode != http.StatusBadRequest {
		t.Fatalf("code not registered: %v %v", c, ok)
	}
}
