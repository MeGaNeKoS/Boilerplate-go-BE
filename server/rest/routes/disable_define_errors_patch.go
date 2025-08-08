//go:build !disable_huma_patch

package routes

import (
	"github.com/bouk/monkey"
	"github.com/danielgtaylor/huma/v2"

	// Required for go:linkname
	_ "unsafe"
)

// Patch: disable Huma's internal defineErrors function by replacing it with a
// no-op via monkey patching. This prevents automatic default error responses and
// related schemas from being registered.
//
// This is a hack relying on go:linkname and runtime patching. It is **unsafe**
// and may break with future Go releases. Build with `-tags=disable_huma_patch`
// to opt out.
//
// ⚠️ Do NOT delete this file unless you are replacing the patch via a safer
// method.
//
//go:linkname defineErrors github.com/danielgtaylor/huma/v2.defineErrors
func defineErrors(op *huma.Operation, registry huma.Registry)

func noopDefineErrors(op *huma.Operation, registry huma.Registry) {
}

func init() {
	monkey.Patch(defineErrors, noopDefineErrors)
}
