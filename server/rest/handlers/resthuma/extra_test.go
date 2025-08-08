package resthuma_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/bouk/monkey"
	"github.com/danielgtaylor/huma/v2/humatest"

	dtoitem "project-template/infrastructure/dto/item"
	"project-template/pkg/code"
	"project-template/server/rest/handlers"
	"project-template/server/rest/handlers/resthuma"
)

func TestSuccessResponseDefaultStatus(t *testing.T) {
	cleanup := withAppName("APP")
	defer cleanup()

	// Call with status 0 to hit the defaulting logic. We don't execute the body
	// to avoid writing an invalid HTTP status code.
	_ = resthuma.SuccessResponse(0, map[string]int{"id": 1})
}

func TestResponseMapVariants(t *testing.T) {
	cleanup := withAppName("APP")
	defer cleanup()

	succ := []resthuma.Success[struct{}]{
		resthuma.NewSuccess(0, "default", struct{}{}),
		resthuma.NewSuccess(http.StatusNoContent, "empty", struct{}{}),
	}

	m := resthuma.ResponseMap[struct{}]("Op", succ, []*code.Code{code.ErrInternalServerError})
	if _, ok := m["200"]; !ok {
		t.Fatalf("missing default 200 response")
	}
	if r, ok := m["204"]; !ok || r.Content != nil {
		t.Fatalf("expected empty 204 response")
	}

	// trigger example value deduplication and duplicate name handling
	resthuma.ResponseMap[struct{}]("OtherOp", succ, nil)
	resthuma.ResponseMap[struct{}]("Op", succ, nil)
}

func TestRegisterExamplesAndSchemas(t *testing.T) {
	cleanup := withAppName("APP")
	defer cleanup()

	succ := []resthuma.Success[dtoitem.Item]{resthuma.NewSuccess(http.StatusOK, "ok", dtoitem.Item{})}
	resthuma.ResponseMap[dtoitem.Item]("ItemOp", succ, nil)
	// Components absent for examples
	_, api := humatest.New(t)
	api.OpenAPI().Components = nil
	resthuma.RegisterExamples(api)

	// Components present but examples absent
	_, api2 := humatest.New(t)
	api2.OpenAPI().Components.Examples = nil
	resthuma.RegisterExamples(api2)
	resthuma.RegisterExamples(api2)

	// Components absent for schemas
	_, api3 := humatest.New(t)
	api3.OpenAPI().Components = nil
	resthuma.RegisterSchemas(api3)

	// Components present but schemas absent
	_, api4 := humatest.New(t)
	api4.OpenAPI().Components.Schemas = nil
	resthuma.RegisterSchemas(api4)
	resthuma.RegisterSchemas(api4)
}

func TestResponseMapFromHandlerExtra(t *testing.T) {
	cleanup := withAppName("APP")
	defer cleanup()

	succ := []resthuma.Success[dtoitem.Item]{resthuma.NewSuccess(http.StatusOK, "ok", dtoitem.Item{})}
	m := resthuma.ResponseMapFromHandler[dtoitem.Item]("GetItem", succ, handlers.GetItem, code.ErrBadRequest)
	if _, ok := m["400"]; !ok {
		t.Fatalf("missing extra error response")
	}
}

func TestInferErrorCodesCacheAndInvalid(t *testing.T) {
	if codes := resthuma.InferErrorCodes(123); codes != nil {
		t.Fatalf("expected nil for non-function input")
	}
	first := resthuma.InferErrorCodes(resthuma.RegisterExamples)
	second := resthuma.InferErrorCodes(resthuma.RegisterExamples)
	if len(first) != len(second) {
		t.Fatalf("cache returned different results")
	}
}

func TestInferErrorCodesNilFunc(t *testing.T) {
	var f func()
	if codes := resthuma.InferErrorCodes(f); codes != nil {
		t.Fatalf("expected nil for nil function")
	}
}

func TestInferErrorCodesNoDot(t *testing.T) {
	var f = func() {}
	// Force the last index check to fail by patching strings.LastIndex.
	patch := monkey.Patch(strings.LastIndex, func(string, string) int { return -1 })
	defer patch.Unpatch()
	if codes := resthuma.InferErrorCodes(f); codes != nil {
		t.Fatalf("expected nil when function name lacks dot")
	}
}

func TestRegisterExampleError(t *testing.T) {
	cleanup := withAppName("APP")
	defer cleanup()

	type bad struct{ C chan int }
	succ := []resthuma.Success[bad]{resthuma.NewSuccess(http.StatusOK, "bad", bad{})}
	resthuma.ResponseMap[bad]("Bad", succ, nil)
	// duplicate to trigger name conflict path even when example cannot be marshaled
	resthuma.ResponseMap[bad]("Bad", succ, nil)
}
