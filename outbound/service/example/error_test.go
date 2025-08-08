package example

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/bouk/monkey"

	"project-template/infrastructure/config"
	"project-template/infrastructure/dto/response"
	"project-template/outbound/transport"
	codepkg "project-template/pkg/code"
	"project-template/pkg/logger"
)

func TestFetchItemByIDAdditionalErrors(t *testing.T) {
	config.Cfg = &config.Config{Service: config.Service{Example: config.ServiceDetail{Host: "h"}}}
	l := stubLogger{parent: "p"}
	o := &exampleOutbound{log: l}

	// wrong type
	monkey.PatchInstanceMethod(reflect.TypeOf(&transport.HTTPOutbound{}), "SendHTTPRequest", func(*transport.HTTPOutbound, logger.Logger) (response.HttpResponse, *codepkg.Code) {
		return response.HttpResponse{RawResponsePayload: struct{}{}}, nil
	})
	_, err := o.FetchItemByID(context.Background(), 1)
	if err == nil || err.Error() != "unexpected response type" {
		t.Fatalf("expected type error, got %v", err)
	}
	monkey.UnpatchAll()

	// marshal error
	monkey.PatchInstanceMethod(reflect.TypeOf(&transport.HTTPOutbound{}), "SendHTTPRequest", func(*transport.HTTPOutbound, logger.Logger) (response.HttpResponse, *codepkg.Code) {
		gr := &response.GenericResponse[any]{}
		gr.Body = struct{}{}
		return response.HttpResponse{RawResponsePayload: gr}, nil
	})
	monkey.Patch(json.Marshal, func(any) ([]byte, error) { return nil, errors.New("m") })
	_, err = o.FetchItemByID(context.Background(), 2)
	if err == nil || err.Error() != "m" {
		t.Fatalf("expected marshal error, got %v", err)
	}
	monkey.Unpatch(json.Marshal)
	monkey.UnpatchAll()

	// unmarshal error
	monkey.PatchInstanceMethod(reflect.TypeOf(&transport.HTTPOutbound{}), "SendHTTPRequest", func(*transport.HTTPOutbound, logger.Logger) (response.HttpResponse, *codepkg.Code) {
		gr := &response.GenericResponse[any]{}
		gr.Body = map[string]any{}
		return response.HttpResponse{RawResponsePayload: gr}, nil
	})
	monkey.Patch(json.Marshal, func(any) ([]byte, error) { return []byte("{}"), nil })
	monkey.Patch(json.Unmarshal, func([]byte, any) error { return errors.New("u") })
	_, err = o.FetchItemByID(context.Background(), 3)
	if err == nil || err.Error() != "u" {
		t.Fatalf("expected unmarshal error, got %v", err)
	}
	monkey.Unpatch(json.Marshal)
	monkey.Unpatch(json.Unmarshal)
	monkey.UnpatchAll()
}

func TestFetchItemByFilterAdditionalErrors(t *testing.T) {
	config.Cfg = &config.Config{Service: config.Service{Example: config.ServiceDetail{Host: "h"}}}
	l := stubLogger{parent: "p"}
	o := &exampleOutbound{log: l}

	// code error
	monkey.PatchInstanceMethod(reflect.TypeOf(&transport.HTTPOutbound{}), "SendHTTPRequest", func(*transport.HTTPOutbound, logger.Logger) (response.HttpResponse, *codepkg.Code) {
		return response.HttpResponse{}, &codepkg.Code{Message: "bad"}
	})
	_, err := o.FetchItemByFilter(context.Background(), "")
	if err == nil || err.Error() != "external request failed: bad" {
		t.Fatalf("expected external error, got %v", err)
	}
	monkey.UnpatchAll()

	// marshal error
	monkey.PatchInstanceMethod(reflect.TypeOf(&transport.HTTPOutbound{}), "SendHTTPRequest", func(*transport.HTTPOutbound, logger.Logger) (response.HttpResponse, *codepkg.Code) {
		gr := &response.GenericResponse[any]{}
		gr.Body = struct{}{}
		return response.HttpResponse{RawResponsePayload: gr}, nil
	})
	monkey.Patch(json.Marshal, func(any) ([]byte, error) { return nil, errors.New("m") })
	_, err = o.FetchItemByFilter(context.Background(), "f")
	if err == nil || err.Error() != "m" {
		t.Fatalf("expected marshal error, got %v", err)
	}
	monkey.Unpatch(json.Marshal)
	monkey.UnpatchAll()

	// unmarshal error
	monkey.PatchInstanceMethod(reflect.TypeOf(&transport.HTTPOutbound{}), "SendHTTPRequest", func(*transport.HTTPOutbound, logger.Logger) (response.HttpResponse, *codepkg.Code) {
		gr := &response.GenericResponse[any]{}
		gr.Body = []any{}
		return response.HttpResponse{RawResponsePayload: gr}, nil
	})
	monkey.Patch(json.Marshal, func(any) ([]byte, error) { return []byte("[]"), nil })
	monkey.Patch(json.Unmarshal, func([]byte, any) error { return errors.New("u") })
	_, err = o.FetchItemByFilter(context.Background(), "f")
	if err == nil || err.Error() != "u" {
		t.Fatalf("expected unmarshal error, got %v", err)
	}
	monkey.Unpatch(json.Marshal)
	monkey.Unpatch(json.Unmarshal)
	monkey.UnpatchAll()
}
