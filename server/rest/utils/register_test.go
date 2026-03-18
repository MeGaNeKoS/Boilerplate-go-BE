package utils

import (
	"context"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

type regInput struct {
	Hidden string `query:"h" internal:"true"`
}

func TestRegisterTracksInternalParams(t *testing.T) {
	ClearInternalParams()
	r := chi.NewRouter()
	cfg := huma.DefaultConfig("test", "1.0")
	cfg.CreateHooks = nil
	api := humachi.New(r, cfg)
	op := huma.Operation{OperationID: "op", Method: http.MethodGet, Path: "/x"}
	Register(api, op, func(ctx context.Context, in *regInput) (*struct{}, error) {
		return nil, nil
	})
	params := InternalParamsFor("op")
	if len(params) != 1 || params[0].Name != "h" || params[0].In != "query" {
		t.Fatalf("unexpected params: %v", params)
	}
}
