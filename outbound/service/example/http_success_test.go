package example

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"github.com/bouk/monkey"

	"project-template/infrastructure/config"
	models "project-template/infrastructure/dto/item"
	"project-template/infrastructure/dto/response"
	"project-template/outbound/transport"
	codepkg "project-template/pkg/code"
	"project-template/pkg/logger"
)

func TestFetchItemByIDSuccess(t *testing.T) {
	config.Cfg = &config.Config{Service: config.Service{Example: config.ServiceDetail{Host: "h"}}}
	l := stubLogger{parent: "p"}
	o := &exampleOutbound{log: l}

	monkey.PatchInstanceMethod(reflect.TypeOf(&transport.HTTPOutbound{}), "SendHTTPRequest", func(_ *transport.HTTPOutbound, _ logger.Logger) (response.HttpResponse, *codepkg.Code) {
		gr := &response.GenericResponse[any]{}
		gr.Body = models.Item{ID: 5, Name: "n"}
		return response.HttpResponse{RawResponsePayload: gr, HTTPCode: http.StatusOK}, nil
	})
	defer monkey.UnpatchAll()

	item, err := o.FetchItemByID(context.Background(), 5)
	if err != nil {
		t.Fatalf("FetchItemByID error: %v", err)
	}
	if item.ID != 5 || item.Name != "n" {
		t.Fatalf("unexpected item %#v", item)
	}
}

func TestFetchItemByFilterSuccess(t *testing.T) {
	config.Cfg = &config.Config{Service: config.Service{Example: config.ServiceDetail{Host: "h"}}}
	l := stubLogger{parent: "p"}
	o := &exampleOutbound{log: l}

	monkey.PatchInstanceMethod(reflect.TypeOf(&transport.HTTPOutbound{}), "SendHTTPRequest", func(_ *transport.HTTPOutbound, _ logger.Logger) (response.HttpResponse, *codepkg.Code) {
		gr := &response.GenericResponse[any]{}
		gr.Body = []models.Item{{ID: 2, Name: "z"}}
		return response.HttpResponse{RawResponsePayload: gr, HTTPCode: http.StatusOK}, nil
	})
	defer monkey.UnpatchAll()

	items, err := o.FetchItemByFilter(context.Background(), "f")
	if err != nil {
		t.Fatalf("FetchItemByFilter error: %v", err)
	}
	if len(items) != 1 || items[0].ID != 2 || items[0].Name != "z" {
		t.Fatalf("unexpected items %#v", items)
	}
}
