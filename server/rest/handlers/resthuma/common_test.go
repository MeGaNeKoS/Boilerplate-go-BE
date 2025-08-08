package resthuma_test

import (
	"testing"

	"project-template/pkg/code"
	"project-template/server/rest/handlers"
	"project-template/server/rest/handlers/resthuma"
)

func containsCode(list []*code.Code, c *code.Code) bool {
	for _, v := range list {
		if v == c {
			return true
		}
	}
	return false
}

func TestInferErrorCodes(t *testing.T) {
	cases := []struct {
		handler  any
		expected []*code.Code
	}{
		{handlers.ListItems, []*code.Code{code.ErrInternalServerError}},
		{handlers.UploadFile, []*code.Code{code.ErrPayloadError}},
		{handlers.GetItem, []*code.Code{code.ErrInternalServerError, code.ErrItemNotFound}},
		{handlers.UpdateItem, []*code.Code{code.ErrInternalServerError}},
		{handlers.DeleteItem, []*code.Code{code.ErrInternalServerError}},
		{resthuma.RegisterExamples, nil},
	}
	for _, c := range cases {
		got := resthuma.InferErrorCodes(c.handler)
		if len(got) < len(c.expected) {
			t.Fatalf("expected at least %d codes, got %d", len(c.expected), len(got))
		}
		for _, e := range c.expected {
			if !containsCode(got, e) {
				t.Fatalf("missing %v", e)
			}
		}
	}
}
