package utils_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/bouk/monkey"
	"github.com/danielgtaylor/huma/v2/humatest"

	dtoitem "project-template/infrastructure/dto/item"
	"project-template/pkg/code"
	"project-template/server/rest/handlers"
	restutils "project-template/server/rest/utils"
)

func TestSuccessResponseDefaultStatus(t *testing.T) {
	cleanup := withAppName("APP")
	defer cleanup()

	// Call with status 0 to hit the defaulting logic. We don't execute the body
	// to avoid writing an invalid HTTP status code.
	_ = restutils.SuccessResponse(0, map[string]int{"id": 1})
}

func TestResponseMapVariants(t *testing.T) {
	cleanup := withAppName("APP")
	defer cleanup()

	succ := restutils.Successes(
		restutils.NewSuccess(0, "default", struct{}{}),
		restutils.NewSuccess(http.StatusNoContent, "empty", struct{}{}),
	)

	m := restutils.ResponseMap("Op", succ, []*code.Code{code.ErrInternalServerError})
	if _, ok := m["200"]; !ok {
		t.Fatalf("missing default 200 response")
	}
	if r, ok := m["204"]; !ok || r.Content != nil {
		t.Fatalf("expected empty 204 response")
	}

	// trigger example value deduplication and duplicate name handling
	restutils.ResponseMap("OtherOp", succ, nil)
	restutils.ResponseMap("Op", succ, nil)
}

func TestRegisterExamplesAndSchemas(t *testing.T) {
	cleanup := withAppName("APP")
	defer cleanup()

	succ := restutils.Successes(restutils.NewSuccess(http.StatusOK, "ok", dtoitem.Item{}))
	restutils.ResponseMap("ItemOp", succ, nil)
	// Components absent for examples
	_, api := humatest.New(t)
	api.OpenAPI().Components = nil
	restutils.RegisterExamples(api)

	// Components present but examples absent
	_, api2 := humatest.New(t)
	api2.OpenAPI().Components.Examples = nil
	restutils.RegisterExamples(api2)
	restutils.RegisterExamples(api2)

	// Components absent for schemas
	_, api3 := humatest.New(t)
	api3.OpenAPI().Components = nil
	restutils.RegisterSchemas(api3)

	// Components present but schemas absent
	_, api4 := humatest.New(t)
	api4.OpenAPI().Components.Schemas = nil
	restutils.RegisterSchemas(api4)
	restutils.RegisterSchemas(api4)
}

func TestResponseMapFromHandlerExtra(t *testing.T) {
	cleanup := withAppName("APP")
	defer cleanup()

	succ := restutils.Successes(restutils.NewSuccess(http.StatusOK, "ok", dtoitem.Item{}))
	m := restutils.ResponseMapFromHandler("GetItem", succ, handlers.GetItem, code.ErrBadRequest)
	if _, ok := m["400"]; !ok {
		t.Fatalf("missing extra error response")
	}
}

func TestInferErrorCodesCacheAndInvalid(t *testing.T) {
	if codes := restutils.InferErrorCodes(123); codes != nil {
		t.Fatalf("expected nil for non-function input")
	}
	first := restutils.InferErrorCodes(restutils.RegisterExamples)
	second := restutils.InferErrorCodes(restutils.RegisterExamples)
	if len(first) != len(second) {
		t.Fatalf("cache returned different results")
	}
}

func TestInferErrorCodesNilFunc(t *testing.T) {
	var f func()
	if codes := restutils.InferErrorCodes(f); codes != nil {
		t.Fatalf("expected nil for nil function")
	}
}

func TestInferErrorCodesNoDot(t *testing.T) {
	var f = func() {}
	// Force the last index check to fail by patching strings.LastIndex.
	patch := monkey.Patch(strings.LastIndex, func(string, string) int { return -1 })
	defer patch.Unpatch()
	if codes := restutils.InferErrorCodes(f); codes != nil {
		t.Fatalf("expected nil when function name lacks dot")
	}
}

func TestRegisterExampleError(t *testing.T) {
	cleanup := withAppName("APP")
	defer cleanup()

	type bad struct{ C chan int }
	succ := restutils.Successes(restutils.NewSuccess(http.StatusOK, "bad", bad{}))
	restutils.ResponseMap("Bad", succ, nil)
	// duplicate to trigger name conflict path even when example cannot be marshaled
	restutils.ResponseMap("Bad", succ, nil)
}
func TestNoSuccesses(t *testing.T) {
	if ns := restutils.NoSuccesses(); ns != nil {
		t.Fatalf("expected nil slice")
	}
}

func TestResponseMapDefaultContentTypeAndAny(t *testing.T) {
	cleanup := withAppName("APP")
	defer cleanup()

	succ := restutils.Successes(
		restutils.NewSuccess(http.StatusOK, "ok", struct{}{}, restutils.WithContentType("")),
	)
	m := restutils.ResponseMap("DefaultCT", succ, nil)
	r, ok := m["200"]
	if !ok {
		t.Fatalf("missing 200 response")
	}
	if _, ok := r.Content["application/json"]; !ok {
		t.Fatalf("expected default application/json content type")
	}
	ref := r.Content["application/json"].Examples["success"].Ref
	if !strings.Contains(ref, "Any-200") {
		t.Fatalf("expected example ref to include 'Any-200', got %s", ref)
	}
}

func TestResponseMapAnyForInterface(t *testing.T) {
	cleanup := withAppName("APP")
	defer cleanup()

	succ := restutils.Successes(
		restutils.NewSuccess[any](http.StatusOK, "ok", nil),
	)
	m := restutils.ResponseMap[any]("InterfaceOp", succ, nil)
	r, ok := m["200"]
	if !ok {
		t.Fatalf("missing 200 response")
	}
	ref := r.Content["application/json"].Examples["success"].Ref
	if !strings.Contains(ref, "Any-200") {
		t.Fatalf("expected example ref to include 'Any-200', got %s", ref)
	}
}

func TestResponseMapAnyNonJSON(t *testing.T) {
	cleanup := withAppName("APP")
	defer cleanup()

	succ := restutils.Successes(
		restutils.NewSuccess(http.StatusOK, "ok", struct{}{}, restutils.WithContentType("text/plain")),
	)
	m := restutils.ResponseMap("PlainOp", succ, nil)
	r, ok := m["200"]
	if !ok {
		t.Fatalf("missing 200 response")
	}
	if _, ok := r.Content["text/plain"]; !ok {
		t.Fatalf("expected text/plain content type")
	}
	ref := r.Content["text/plain"].Examples["success"].Ref
	if !strings.Contains(ref, "PlainOp-Any-200") {
		t.Fatalf("expected example ref to include 'PlainOp-Any-200', got %s", ref)
	}
}
