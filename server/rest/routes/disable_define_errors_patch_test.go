package routes

import (
	"context"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	humachi "github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

func TestDefineErrorsPatched(t *testing.T) {
	r := chi.NewRouter()
	api := humachi.New(r, huma.DefaultConfig("x", "1"))
	huma.Get(api, "/test", func(ctx context.Context, in *struct{}) (*struct{ Message string }, error) {
		return &struct{ Message string }{Message: "ok"}, nil
	})
	spec := api.OpenAPI()
	resps := spec.Paths["/test"].Get.Responses
	if _, ok := resps["400"]; ok {
		t.Fatalf("unexpected default 400 response: %+v", resps)
	}
	if spec.Components != nil && spec.Components.Schemas != nil {
		m := spec.Components.Schemas.Map()
		if _, ok := m["ErrorModel"]; ok {
			t.Fatalf("unexpected ErrorModel schema present")
		}
		if _, ok := m["ErrorDetail"]; ok {
			t.Fatalf("unexpected ErrorDetail schema present")
		}
	}
}
