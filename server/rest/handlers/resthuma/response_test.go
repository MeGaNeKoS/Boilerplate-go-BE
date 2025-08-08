package resthuma_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"

	"project-template/infrastructure/config"
	dtoitem "project-template/infrastructure/dto/item"
	"project-template/infrastructure/dto/response"
	"project-template/pkg/code"
	"project-template/server/rest/handlers"
	"project-template/server/rest/handlers/resthuma"
)

func withAppName(name string) func() {
	prev := config.Cfg
	config.Cfg = &config.Config{AppName: name}
	return func() { config.Cfg = prev }
}

func TestNewError(t *testing.T) {
	cleanup := withAppName("APP")
	defer cleanup()

	err := resthuma.NewError(code.ErrItemNotFound)
	he, ok := err.(interface {
		GetStatus() int
		GetHeaders() http.Header
	})
	if !ok {
		t.Fatalf("returned error does not implement expected interface")
	}
	if he.GetStatus() != code.ErrItemNotFound.HTTPCode {
		t.Fatalf("expected status %d, got %d", code.ErrItemNotFound.HTTPCode, he.GetStatus())
	}
	if ct := he.GetHeaders().Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("expected problem+json content type, got %s", ct)
	}
	if err.Error() != code.ErrItemNotFound.Message {
		t.Fatalf("unexpected error message: %s", err.Error())
	}

	err = resthuma.NewError(code.ErrItemNotFound, "custom")
	if err.Error() != "custom" {
		t.Fatalf("expected overridden message, got %s", err.Error())
	}
}

func TestResponseHeadersAndBody(t *testing.T) {
	r := resthuma.NewResponse(http.StatusCreated, map[string]string{"foo": "bar"})
	r.Header("X-Test", "123")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp := httptest.NewRecorder()
	ctx := humatest.NewContext(&huma.Operation{}, req, resp)

	r.Body(ctx)

	if resp.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, resp.Code)
	}
	if resp.Header().Get("X-Test") != "123" {
		t.Fatalf("header not set on response")
	}
	if resp.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("expected default content type")
	}
	var body map[string]string
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["foo"] != "bar" {
		t.Fatalf("unexpected body: %#v", body)
	}
	if r.GetHeaders().Get("X-Test") != "123" {
		t.Fatalf("header not stored in struct")
	}
}

func TestSuccessAndNoContentResponses(t *testing.T) {
	cleanup := withAppName("APP")
	defer cleanup()

	s := resthuma.SuccessResponse(http.StatusOK, map[string]int{"id": 1})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp := httptest.NewRecorder()
	ctx := humatest.NewContext(&huma.Operation{}, req, resp)
	s.Body(ctx)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}
	var gr response.GenericResponse[map[string]int]
	if err := json.Unmarshal(resp.Body.Bytes(), &gr); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if gr.StatusCode != "APP-200" || gr.Body["id"] != 1 {
		t.Fatalf("unexpected envelope: %#v", gr)
	}

	nc := resthuma.NoContentResponse()
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	resp2 := httptest.NewRecorder()
	ctx2 := humatest.NewContext(&huma.Operation{}, req2, resp2)
	nc.Body(ctx2)
	if resp2.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, resp2.Code)
	}
	if strings.TrimSpace(resp2.Body.String()) != "{}" {
		t.Fatalf("unexpected body %q", resp2.Body.String())
	}
}

func TestResponseMapAndRegister(t *testing.T) {
	cleanup := withAppName("APP")
	defer cleanup()

	successes := []resthuma.Success[dtoitem.Item]{
		resthuma.NewSuccess(http.StatusCreated, "created", dtoitem.Item{ID: 1, Name: "x"}),
	}
	m := resthuma.ResponseMap[dtoitem.Item]("op", successes, []*code.Code{code.ErrInternalServerError})
	if _, ok := m["201"]; !ok {
		t.Fatalf("missing success response")
	}
	if _, ok := m["500"]; !ok {
		t.Fatalf("missing error response")
	}

	_, api := humatest.New(t)
	resthuma.RegisterExamples(api)
	resthuma.RegisterSchemas(api)
	if len(api.OpenAPI().Components.Examples) == 0 {
		t.Fatalf("expected examples to be registered")
	}
	if api.OpenAPI().Components.Schemas == nil || len(api.OpenAPI().Components.Schemas.Map()) == 0 {
		t.Fatalf("expected schemas to be registered")
	}
}

func TestResponseMapFromHandler(t *testing.T) {
	cleanup := withAppName("APP")
	defer cleanup()

	succ := []resthuma.Success[dtoitem.Item]{resthuma.NewSuccess(http.StatusOK, "ok", dtoitem.Item{})}
	m := resthuma.ResponseMapFromHandler[dtoitem.Item]("GetItem", succ, handlers.GetItem)
	if _, ok := m["500"]; !ok {
		t.Fatalf("expected inferred error response")
	}
	if _, ok := m["404"]; !ok {
		t.Fatalf("expected not found error response")
	}
}
func TestResponseMultiHeaderAndBytes(t *testing.T) {
	r := resthuma.NewResponse(http.StatusOK, []byte("raw"))
	r.Header("X-Test", "a").Header("X-Test", "b")
	r.Header("Content-Type", "text/plain")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp := httptest.NewRecorder()
	ctx := humatest.NewContext(&huma.Operation{}, req, resp)

	r.Body(ctx)

	values := resp.Header()["X-Test"]
	if len(values) != 2 || values[0] != "a" || values[1] != "b" {
		t.Fatalf("expected appended headers, got %v", values)
	}
	if ct := resp.Header().Get("Content-Type"); ct != "text/plain" {
		t.Fatalf("expected content type preserved, got %s", ct)
	}
	if body := resp.Body.String(); body != "raw" {
		t.Fatalf("expected raw body, got %q", body)
	}
}
