package utils

import (
	"net/http"
	"testing"

	"project-template/infrastructure/config"
	"project-template/infrastructure/dto/response"
	"project-template/pkg/code"
)

func TestGenerateErrorResponse(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	err := code.Code{HTTPCode: http.StatusBadRequest, InternalCode: 123, Message: "bad"}
	resp := GenerateErrorResponse(err)
	if resp.HTTPCode != http.StatusBadRequest {
		t.Fatalf("HTTPCode got %d want %d", resp.HTTPCode, http.StatusBadRequest)
	}
	generic, ok := resp.RawResponsePayload.(response.GenericResponse)
	if !ok {
		t.Fatalf("unexpected payload type %T", resp.RawResponsePayload)
	}
	expectedStatus := "APP-ERR-123"
	if generic.StatusCode != expectedStatus {
		t.Errorf("status code got %q want %q", generic.StatusCode, expectedStatus)
	}
	if generic.ReturnMessage != "bad" {
		t.Errorf("message got %q want %q", generic.ReturnMessage, "bad")
	}
}

func TestGenerateErrorResponseOverride(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	err := code.Code{HTTPCode: http.StatusBadRequest, InternalCode: 1, Message: "bad"}
	resp := GenerateErrorResponse(err, "oops")
	g := resp.RawResponsePayload.(response.GenericResponse)
	if g.ReturnMessage != "oops" {
		t.Fatalf("override failed")
	}
}

func TestGenerateSuccessResponse(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	body := map[string]string{"x": "y"}
	resp := GenerateSuccessResponse(body)
	if resp.HTTPCode != http.StatusOK {
		t.Fatalf("HTTPCode got %d want %d", resp.HTTPCode, http.StatusOK)
	}
	generic, ok := resp.RawResponsePayload.(response.GenericResponse)
	if !ok {
		t.Fatalf("unexpected payload type %T", resp.RawResponsePayload)
	}
	if generic.StatusCode != "APP-200" {
		t.Errorf("status code got %q want %q", generic.StatusCode, "APP-200")
	}
	if generic.ReturnMessage != "Success" {
		t.Errorf("message got %q want %q", generic.ReturnMessage, "Success")
	}
	gotBody, ok := generic.Body.(map[string]string)
	if !ok || gotBody["x"] != "y" {
		t.Errorf("body got %#v", generic.Body)
	}
}

func TestGenerateSuccessResponseOpts(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	code := 201
	msg := "created"
	resp := GenerateSuccessResponse(nil, &SuccessOptions{HTTPCode: &code, Message: &msg})
	if resp.HTTPCode != 201 {
		t.Fatalf("HTTPCode got %d want 201", resp.HTTPCode)
	}
	generic := resp.RawResponsePayload.(response.GenericResponse)
	if generic.ReturnMessage != msg {
		t.Errorf("message got %q want %q", generic.ReturnMessage, msg)
	}
}
