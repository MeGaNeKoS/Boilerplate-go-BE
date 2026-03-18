package routes

import (
	"strings"

	"github.com/danielgtaylor/huma/v2"
)

const BearerScheme = "bearerAuth"

// NewGroup wraps huma.NewGroup and applies a default tag based on the prefix.
func NewGroup(parent *huma.Group, prefix string) *huma.Group {
	var g *huma.Group
	if prefix != "" {
		g = huma.NewGroup(parent, prefix)
	} else {
		g = huma.NewGroup(parent)
	}
	UseDefaultTag(g, prefix)
	return g
}

// UseDefaultTag sets a default tag for operations based on the group prefix.
// If an operation already specifies tags, they are left unchanged.
func UseDefaultTag(g *huma.Group, prefix string) {
	tag := strings.Trim(strings.TrimPrefix(prefix, "/"), "/")
	if tag == "" {
		return
	}
	g.UseSimpleModifier(func(o *huma.Operation) {
		if len(o.Tags) == 0 {
			o.Tags = []string{tag}
		}
	})
}

// UseSecurity sets a default security requirement for operations in the group
// if none are already specified.
func UseSecurity(g *huma.Group, scheme string) {
	g.UseSimpleModifier(func(o *huma.Operation) {
		if len(o.Security) == 0 {
			o.Security = []map[string][]string{{scheme: {}}}
		}
	})
}
