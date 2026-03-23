package code

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestCodeMessageHelpers(t *testing.T) {
	c := Code{HTTPCode: http.StatusBadRequest, Message: "bad"}
	if got := c.PrependMessage("first"); got.Message != "first: bad" || got.HTTPCode != http.StatusBadRequest {
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

func TestCodeStatusCode(t *testing.T) {
	c := Code{HTTPCode: http.StatusNotFound, Message: "not found"}
	if c.StatusCode() != http.StatusNotFound {
		t.Fatalf("unexpected status %d", c.StatusCode())
	}
}

func TestCodeMarshalJSON(t *testing.T) {
	c := Code{HTTPCode: http.StatusBadRequest, Message: "bad request"}
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	expected := `{"status":400,"message":"bad request"}`
	if string(b) != expected {
		t.Fatalf("unexpected JSON: %s", b)
	}
}
