package helpers

// hideFromPublicTag is applied to operations that should be omitted from the
// public OpenAPI specification. The actual value is unlikely to collide with
// user-defined tags.
const hideFromPublicTag = "_hide_from_public_api"

// InternalTag returns the tag used to mark private operations.
func InternalTag() string {
	return hideFromPublicTag
}
