package response

import "testing"

func TestWithHeaderInitializesMap(t *testing.T) {
	r := &HttpResponse{}
	got := r.WithHeader("Content-Type", "application/json")
	if got != r {
		t.Fatal("WithHeader should return the same pointer")
	}
	if r.Headers == nil {
		t.Fatal("Headers map should be initialized")
	}
	if r.Headers["Content-Type"] != "application/json" {
		t.Fatalf("unexpected header value: %s", r.Headers["Content-Type"])
	}
}

func TestWithHeaderExistingMap(t *testing.T) {
	r := &HttpResponse{Headers: map[string]string{"Existing": "value"}}
	r.WithHeader("New", "val")
	if r.Headers["Existing"] != "value" {
		t.Fatal("existing header lost")
	}
	if r.Headers["New"] != "val" {
		t.Fatal("new header not set")
	}
}

func TestWithHeaderOverwrite(t *testing.T) {
	r := &HttpResponse{}
	r.WithHeader("K", "v1").WithHeader("K", "v2")
	if r.Headers["K"] != "v2" {
		t.Fatalf("expected overwritten value v2, got %s", r.Headers["K"])
	}
}

func TestWithInstanceProblemDetail(t *testing.T) {
	pd := ProblemDetail{Title: "err", Status: 400}
	r := &HttpResponse{RawResponsePayload: pd}
	got := r.WithInstance("/api/test")
	if got != r {
		t.Fatal("WithInstance should return the same pointer")
	}
	updated, ok := r.RawResponsePayload.(ProblemDetail)
	if !ok {
		t.Fatal("payload should still be ProblemDetail")
	}
	if updated.Instance != "/api/test" {
		t.Fatalf("expected instance /api/test, got %s", updated.Instance)
	}
}

func TestWithInstanceNonProblemDetail(t *testing.T) {
	r := &HttpResponse{RawResponsePayload: "plain string"}
	got := r.WithInstance("/api/test")
	if got != r {
		t.Fatal("WithInstance should return the same pointer")
	}
	if r.RawResponsePayload != "plain string" {
		t.Fatal("payload should be unchanged for non-ProblemDetail")
	}
}

func TestWithInstanceNilPayload(t *testing.T) {
	r := &HttpResponse{}
	got := r.WithInstance("/api/test")
	if got != r {
		t.Fatal("WithInstance should return the same pointer")
	}
	if r.RawResponsePayload != nil {
		t.Fatal("nil payload should remain nil")
	}
}

func TestGenericResponse(t *testing.T) {
	resp := GenericResponse[string]{
		BaseResponse: BaseResponse{StatusCode: "200", ReturnMessage: "OK"},
		Body:         "hello",
	}
	if resp.StatusCode != "200" || resp.ReturnMessage != "OK" || resp.Body != "hello" {
		t.Fatalf("GenericResponse fields mismatch")
	}
}

func TestHttpResponseFields(t *testing.T) {
	r := HttpResponse{
		HTTPCode:           200,
		ContentType:        "text/plain",
		RawResponsePayload: "ok",
	}
	if r.HTTPCode != 200 || r.ContentType != "text/plain" || r.RawResponsePayload != "ok" {
		t.Fatal("field mismatch")
	}
}
