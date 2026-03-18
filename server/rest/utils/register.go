package utils

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
)

// Register wraps huma.Register and records any input parameters marked as
// internal so they can be stripped from the public OpenAPI document.
func Register[I, O any](api huma.API, op huma.Operation, handler func(context.Context, *I) (*O, error)) {
	TrackInternalParams(op.OperationID, (*I)(nil))
	RegisterSchemas(api)
	huma.Register(api, op, handler)
}
