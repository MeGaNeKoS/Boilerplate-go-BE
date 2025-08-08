package utils

import (
	"net/http"
	"reflect"
	"testing"

	"project-template/infrastructure/config"
	"project-template/infrastructure/dto/response"
	"project-template/pkg/code"
)

func TestGenerateErrorResponse(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	err := &code.Code{HTTPCode: http.StatusBadRequest, InternalCode: 123, Message: "bad"}
	resp := GenerateErrorResponse(err)
	if resp.HTTPCode != http.StatusBadRequest {
		t.Fatalf("HTTPCode got %d want %d", resp.HTTPCode, http.StatusBadRequest)
	}
	pd, ok := resp.RawResponsePayload.(response.ProblemDetail)
	if !ok {
		t.Fatalf("unexpected payload type %T", resp.RawResponsePayload)
	}
	expectedCode := "APP-ERR-123"
	if pd.Code != expectedCode {
		t.Errorf("code got %q want %q", pd.Code, expectedCode)
	}
	if pd.Title != "bad" {
		t.Errorf("title got %q want %q", pd.Title, "bad")
	}
	if resp.ContentType != "application/problem+json" {
		t.Errorf("content type %q", resp.ContentType)
	}
}

func TestGenerateErrorResponseOverride(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	err := &code.Code{HTTPCode: http.StatusBadRequest, InternalCode: 1, Message: "bad"}
	resp := GenerateErrorResponse(err, "oops")
	pd := resp.RawResponsePayload.(response.ProblemDetail)
	if pd.Title != "oops" {
		t.Fatalf("override failed")
	}
}

func TestGenerateResponse(t *testing.T) {
	body := map[string]string{"ok": "true"}
	resp := GenerateResponse(http.StatusCreated, body, "")
	if resp.HTTPCode != http.StatusCreated {
		t.Fatalf("status got %d want %d", resp.HTTPCode, http.StatusCreated)
	}
	if resp.ContentType != "application/json" {
		t.Fatalf("default content type got %q", resp.ContentType)
	}
	got, ok := resp.RawResponsePayload.(map[string]string)
	if !ok || !reflect.DeepEqual(got, body) {
		t.Fatalf("payload mismatch")
	}
}

func TestGenerateResponseCustomType(t *testing.T) {
	payload := []byte("data")
	resp := GenerateResponse(http.StatusOK, payload, "text/plain")
	if resp.ContentType != "text/plain" {
		t.Fatalf("content type got %q", resp.ContentType)
	}
	b, ok := resp.RawResponsePayload.([]byte)
	if !ok || string(b) != "data" {
		t.Fatalf("payload mismatch")
	}
}
