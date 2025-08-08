package resthuma

import (
	"context"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	humachi "github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"project-template/server/rest/helpers"
)

type regInput struct {
	Hidden string `query:"h" internal:"true"`
}

func TestRegisterTracksInternalParams(t *testing.T) {
	helpers.ClearInternalParams()
	r := chi.NewRouter()
	api := humachi.New(r, huma.DefaultConfig("test", "1.0"))
	op := huma.Operation{OperationID: "op", Method: http.MethodGet, Path: "/x"}
	Register[regInput, struct{}](api, op, func(ctx context.Context, in *regInput) (*struct{}, error) {
		return nil, nil
	})
	params := helpers.InternalParamsFor("op")
	if len(params) != 1 || params[0].Name != "h" || params[0].In != "query" {
		t.Fatalf("unexpected params: %v", params)
	}
}
