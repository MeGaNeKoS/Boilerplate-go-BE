package resthuma

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"project-template/server/rest/helpers"
)

// Register wraps huma.Register and records any input parameters marked as
// internal so they can be stripped from the public OpenAPI document.
func Register[I, O any](api huma.API, op huma.Operation, handler func(context.Context, *I) (*O, error)) {
	helpers.TrackInternalParams(op.OperationID, (*I)(nil))
	huma.Register(api, op, handler)
}
