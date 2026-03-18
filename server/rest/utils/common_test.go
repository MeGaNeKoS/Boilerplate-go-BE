package utils_test

import (
	"errors"
	"testing"

	"project-template/pkg/code"
	"project-template/server/rest/handlers"
	restutils "project-template/server/rest/utils"
)

func containsCode(list []*code.Code, c *code.Code) bool {
	for _, v := range list {
		if errors.Is(v, c) {
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
		{restutils.RegisterExamples, nil},
	}
	for _, c := range cases {
		got := restutils.InferErrorCodes(c.handler)
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
