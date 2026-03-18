package utils

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"project-template/infrastructure/config"
	"project-template/infrastructure/dto/response"
)

// RewriteSchemaExamples previously rewrote `$schema` examples but now simply
// removes any leftover `$schema` properties.
func RewriteSchemaExamples(o *huma.OpenAPI, _ string) {
	if o == nil || o.Components == nil || o.Components.Schemas == nil {
		return
	}
	for _, s := range o.Components.Schemas.Map() {
		if s == nil {
			continue
		}
		delete(s.Properties, "$schema")
		for _, sub := range s.AllOf {
			if sub != nil {
				delete(sub.Properties, "$schema")
			}
		}
	}
}

// RewriteSchemaLinks updates Link header examples to point at either the
// internal or public schema URLs depending on the provided flag.
func RewriteSchemaLinks(o *huma.OpenAPI, internal bool) {
	if o == nil || o.Paths == nil {
		return
	}
	for _, item := range o.Paths {
		ops := []*huma.Operation{item.Delete, item.Get, item.Head, item.Options, item.Patch, item.Post, item.Put, item.Trace}
		for _, op := range ops {
			if op == nil || op.Responses == nil {
				continue
			}
			for _, r := range op.Responses {
				if r == nil || r.Headers == nil {
					continue
				}
				h, ok := r.Headers["Link"]
				if !ok || h == nil || h.Examples == nil {
					continue
				}
				var ref string
				for _, mt := range r.Content {
					if mt != nil && mt.Schema != nil && mt.Schema.Ref != "" {
						ref = mt.Schema.Ref
						break
					}
				}
				if ref == "" {
					continue
				}
				link := buildSchemaLink(ref, internal)
				if ex, ok := h.Examples["schema"]; ok && ex != nil {
					ex.Value = link
				}
			}
		}
	}
}

// RewriteExampleNames replaces generated example identifiers under response
// media types with their original human-readable titles. When multiple
// examples share the same title, the internal error code is appended to keep
// the keys unique.
func RewriteExampleNames(o *huma.OpenAPI) {
	if o == nil || o.Paths == nil || o.Components == nil || o.Components.Examples == nil {
		return
	}

	type meta struct{ title, code string }
	lookup := map[string]meta{}
	for name, ex := range o.Components.Examples {
		if ex == nil || ex.Value == nil {
			continue
		}
		if pd, ok := ex.Value.(*response.ProblemDetail); ok && pd != nil {
			lookup[name] = meta{pd.Title, strings.TrimPrefix(pd.Type, "/errors/")}
		}
	}

	for _, item := range o.Paths {
		ops := []*huma.Operation{item.Delete, item.Get, item.Head, item.Options, item.Patch, item.Post, item.Put, item.Trace}
		for _, op := range ops {
			if op == nil || op.Responses == nil {
				continue
			}
			for _, r := range op.Responses {
				if r == nil || r.Content == nil {
					continue
				}
				for _, mt := range r.Content {
					if mt == nil || len(mt.Examples) == 0 {
						continue
					}
					type entry struct {
						orig  string
						ex    *huma.Example
						title string
						code  string
					}
					entries := make([]entry, 0)
					counts := map[string]int{}
					for k, ex := range mt.Examples {
						ref := strings.TrimPrefix(ex.Ref, "#/components/examples/")
						if m, ok := lookup[ref]; ok && m.title != "" {
							counts[m.title]++
							entries = append(entries, entry{k, ex, m.title, m.code})
						} else {
							entries = append(entries, entry{k, ex, "", ""})
						}
					}
					mt.Examples = map[string]*huma.Example{}
					for _, e := range entries {
						name := e.orig
						if e.title != "" {
							name = e.title
							if counts[e.title] > 1 && e.code != "" {
								name = fmt.Sprintf("%s - %s", e.title, e.code)
							}
						}
						mt.Examples[name] = e.ex
					}
				}
			}
		}
	}
}

var privateBlocks []*net.IPNet

// SetPrivateCIDRs configures the list of CIDR blocks treated as internal
// network ranges.
func SetPrivateCIDRs(cidrs []string) {
	privateBlocks = privateBlocks[:0]
	for _, c := range cidrs {
		if _, n, err := net.ParseCIDR(c); err == nil {
			privateBlocks = append(privateBlocks, n)
		}
	}
}

func isPrivate(ctx huma.Context) bool {
	if ctx == nil {
		return false
	}
	addr := ctx.Header("X-Forwarded-For")
	if addr != "" {
		if i := strings.Index(addr, ","); i >= 0 {
			addr = strings.TrimSpace(addr[:i])
		}
	} else if xri := ctx.Header("X-Real-IP"); xri != "" {
		addr = strings.TrimSpace(xri)
	} else {
		addr = ctx.RemoteAddr()
	}
	host, _, err := net.SplitHostPort(addr)
	if err == nil {
		addr = host
	}
	ip := net.ParseIP(addr)
	if ip == nil {
		return false
	}
	for _, b := range privateBlocks {
		if b.Contains(ip) {
			return true
		}
	}
	return false
}

// schemaLinkFromRef builds a Link header value pointing to the JSON schema
// referenced by the given component ref, selecting internal or public paths
// based on the request context.
func schemaLinkFromRef(ctx huma.Context, ref string) string {
	return buildSchemaLink(ref, isPrivate(ctx))
}

func buildSchemaLink(ref string, internal bool) string {
	if ref == "" {
		return ""
	}
	base := ""
	docs := "/docs/public"
	schemaPart := "/schema/v1"
	if config.Cfg != nil {
		pubHost := strings.TrimSuffix(config.Cfg.OpenAPI.Servers.Public, "/")
		if pubHost == "" {
			if h := config.Cfg.REST.Host; h != "" {
				pubHost = fmt.Sprintf("//%s:%s", h, config.Cfg.REST.Port)
			}
		}
		base = pubHost
		if internal {
			if cfg := strings.TrimSuffix(config.Cfg.OpenAPI.Servers.Internal, "/"); cfg != "" {
				base = cfg
			}
			if config.Cfg.OpenAPI.Docs.Internal.URL != "" {
				docs = config.Cfg.OpenAPI.Docs.Internal.URL
			}
			if config.Cfg.OpenAPI.Docs.Internal.Schema != "" {
				schemaPart = config.Cfg.OpenAPI.Docs.Internal.Schema
			}
		} else {
			if config.Cfg.OpenAPI.Docs.Public.URL != "" {
				docs = config.Cfg.OpenAPI.Docs.Public.URL
			}
			if config.Cfg.OpenAPI.Docs.Public.Schema != "" {
				schemaPart = config.Cfg.OpenAPI.Docs.Public.Schema
			}
		}
	}
	path := strings.TrimSuffix(docs+schemaPart, "/")
	docs = strings.TrimSuffix(docs, "/")
	name := strings.TrimPrefix(ref, "#/components/schemas/")
	schema := fmt.Sprintf("<%s%s/%s.json>; rel=\"describedby\"; type=\"application/schema+json\"", base, path, name)
	api := fmt.Sprintf("<%s%s/openapi.json>; rel=\"service-desc\"", base, docs)
	return schema + ", " + api
}

// SchemaLinkTransformer ensures error responses include a Link header pointing
// to the appropriate schema, selecting public or internal docs based on the
// request's origin.
func SchemaLinkTransformer(ctx huma.Context, status string, v any) (any, error) {
	code, err := strconv.Atoi(status)
	if err != nil || code < 400 {
		return v, nil
	}
	if link := schemaLinkFromRef(ctx, problemDetailSchema.Ref); link != "" {
		ctx.SetHeader("Link", link)
	}
	if m, ok := v.(map[string]any); ok {
		if inst, exists := m["instance"]; !exists || inst == nil || inst == "" {
			m["instance"] = ctx.URL().Path
		}
	}
	return v, nil
}

// addSchemaHeader documents a Link header on the response pointing to the
// schema describing its body.
func addSchemaHeader(r *huma.Response, schema *huma.Schema) {
	if schema == nil || schema.Ref == "" {
		return
	}
	link := buildSchemaLink(schema.Ref, false)
	if r.Headers == nil {
		r.Headers = map[string]*huma.Param{}
	}
	r.Headers["Link"] = &huma.Param{
		Description: "schema describing this response",
		Schema:      &huma.Schema{Type: huma.TypeString},
		Examples: map[string]*huma.Example{
			"schema": {Value: link},
		},
	}
}
