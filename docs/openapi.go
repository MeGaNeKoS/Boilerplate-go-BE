//go:generate go run ../cmd/gen_docs/main.go
package docs

import (
	"fmt"
	"net/http"
	"path"
	"strings"

	"project-template/infrastructure/config"
	response "project-template/infrastructure/dto/response"
	"project-template/infrastructure/utils"
	"project-template/server/rest/routes"

	"github.com/swaggest/openapi-go"
	"github.com/swaggest/openapi-go/openapi3"
	swgui "github.com/swaggest/swgui/v3"
)

var (
	// Spec holds the generated OpenAPI specification.
	Spec          openapi3.Spec
	specJSONBytes []byte
	specYAMLBytes []byte
)

func addErrorResponses(r *openapi3.Reflector, oc openapi.OperationContext, defs ...routes.ResponseDef) {
	byStatus := map[int][]routes.ResponseDef{}
	for _, d := range defs {
		status := d.Status
		if status == 0 && d.ErrCode != nil {
			status = d.ErrCode.HTTPCode
		}
		if status == 0 {
			status = http.StatusInternalServerError
		}
		byStatus[status] = append(byStatus[status], d)
	}

	for status, list := range byStatus {
		statusCopy := status
		listCopy := list
		first := listCopy[0]
		model := first.Model
		if model == nil {
			model = new(response.GenericResponse)
		}
		cType := first.ContentType
		if cType == "" {
			cType = "application/json"
		}
		desc := first.Description

		oc.AddRespStructure(model,
			openapi.WithHTTPStatus(statusCopy),
			openapi.WithContentType(cType),
			func(cu *openapi.ContentUnit) {
				if desc != "" {
					cu.Description = desc
				}
			},
			openapi.WithCustomize(func(cor openapi.ContentOrReference) {
				ror, ok := cor.(*openapi3.ResponseOrRef)
				if !ok || ror.Response == nil {
					return
				}

				mt := ror.Response.Content[cType]
				if mt.Examples == nil {
					mt.Examples = map[string]openapi3.ExampleOrRef{}
				}

				appName := "example"
				if config.Cfg != nil && config.Cfg.AppName != "" {
					appName = config.Cfg.AppName
				}

				for _, d := range listCopy {
					if d.ErrCode == nil {
						continue
					}
					c := d.ErrCode
					name := fmt.Sprintf("err_%d", c.InternalCode)

					example := openapi3.ExampleOrRef{Example: (&openapi3.Example{}).
						WithSummary(c.Message).
						WithValue(response.GenericResponse{
							BaseResponse: response.BaseResponse{
								StatusCode:    fmt.Sprintf("%s-ERR-%d", appName, c.InternalCode),
								ReturnMessage: c.Message,
							},
						})}

					r.Spec.ComponentsEns().ExamplesEns().WithMapOfExampleOrRefValuesItem(name, example)

					mt.Examples[name] = openapi3.ExampleOrRef{ExampleReference: (&openapi3.ExampleReference{}).
						WithRef("#/components/examples/" + name)}
				}

				ror.Response.Content[cType] = mt
			}))
	}
}

func addResponses(r *openapi3.Reflector, oc openapi.OperationContext, defs ...routes.ResponseDef) {
	var success []routes.ResponseDef
	var errors []routes.ResponseDef
	for _, d := range defs {
		if d.ErrCode != nil {
			errors = append(errors, d)
		} else {
			success = append(success, d)
		}
	}

	for _, resp := range success {
		status := resp.Status
		if status == 0 {
			status = http.StatusOK
		}
		opts := []openapi.ContentOption{openapi.WithHTTPStatus(status)}
		if resp.ContentType != "" {
			opts = append(opts, openapi.WithContentType(resp.ContentType))
		}
		if resp.Description != "" {
			opts = append(opts, func(cu *openapi.ContentUnit) { cu.Description = resp.Description })
		}
		oc.AddRespStructure(resp.Model, opts...)
	}

	if len(errors) > 0 {
		addErrorResponses(r, oc, errors...)
	}
}

// SpecBytes returns the generated specification in JSON form.
func SpecBytes() []byte { return specJSONBytes }

// YAMLSpecBytes returns the generated specification in YAML form.
func YAMLSpecBytes() []byte { return specYAMLBytes }

// BuildSpec constructs the OpenAPI specification using optional configuration.
// It must be called after configuration is loaded so that server information can
// be applied.
func BuildSpec(cfg *config.Config) {
	r := openapi3.NewReflector()
	r.Spec.Info.Title = "Project Template API"
	r.Spec.Info.WithDescription("Service for managing items and system utilities.")
	version := "0.1rc"
	if cfg != nil && cfg.Version != "" {
		version = cfg.Version
	}
	r.Spec.Info.Version = version
	r.Spec.WithTags(
		*new(openapi3.Tag).WithName("items").WithDescription("Operations about items"),
		*new(openapi3.Tag).WithName("system").WithDescription("System operations"),
		*new(openapi3.Tag).WithName("files").WithDescription("File upload and download"),
	)

	// Define security scheme for authenticated endpoints.
	const secName = "bearerAuth"
	r.Spec.SetHTTPBearerTokenSecurity(secName, "JWT", "Authentication with JWT")

	addRoute := func(prefix string, rt routes.RouteDef) {
		// join the route prefix with the pattern to avoid duplicate slashes
		fullPath := path.Join(prefix, rt.Pattern)
		if !strings.HasPrefix(fullPath, "/") {
			fullPath = "/" + fullPath
		}
		oc, _ := r.NewOperationContext(rt.Method, fullPath)
		if rt.Tag != "" {
			oc.SetTags(rt.Tag)
		}
		if rt.OperationID != "" {
			oc.SetID(rt.OperationID)
		}
		if rt.Summary != "" {
			oc.SetSummary(rt.Summary)
		}
		if rt.Description != "" {
			oc.SetDescription(rt.Description)
		}
		if rt.Auth {
			oc.AddSecurity(secName)
		}
		if rt.Req != nil {
			opts := []openapi.ContentOption{}
			if rt.Req.ContentType != "" {
				opts = append(opts, openapi.WithContentType(rt.Req.ContentType))
			}
			if rt.Req.Description != "" {
				opts = append(opts, func(cu *openapi.ContentUnit) { cu.Description = rt.Req.Description })
			}
			oc.AddReqStructure(rt.Req.Model, opts...)
		}
		if len(rt.Responses) > 0 {
			addResponses(r, oc, rt.Responses...)
		}
		_ = r.AddOperation(oc)
	}

	for _, group := range routes.RouteGroups {
		for _, rt := range *group.Routes {
			addRoute(group.Prefix, rt)
		}
	}

	hostname := "http://localhost:5090"
	basePath := "v1"
	if cfg != nil {
		if cfg.REST.Host != "" && cfg.REST.Port != "" {
			hostname = fmt.Sprintf("http://%s:%s", cfg.REST.Host, cfg.REST.Port)
		}
		normalized := utils.NormalizeBasePath(cfg.Server.Endpoint.Based)
		if normalized != "/" {
			basePath = strings.Trim(normalized, "/")
		}
	}

	r.Spec.Servers = []openapi3.Server{
		*new(openapi3.Server).WithURL("{hostname}/{basePath}").WithVariables(map[string]openapi3.ServerVariable{
			"hostname": *new(openapi3.ServerVariable).WithDescription("The hostname of the API server.").WithDefault(hostname).WithEnum(hostname),
			"basePath": *new(openapi3.ServerVariable).WithDescription("The base path for the API version.").WithDefault(basePath).WithEnum("v1", "beta"),
		}),
		*new(openapi3.Server).WithURL("{customHostname}").WithDescription("Custom deployment server.").WithVariables(map[string]openapi3.ServerVariable{
			"customHostname": *new(openapi3.ServerVariable).WithDescription("Custom hostname with full URL.").WithDefault(hostname),
		}),
	}

	Spec = *r.Spec
	specJSONBytes, _ = Spec.MarshalJSON()
	specYAMLBytes, _ = Spec.MarshalYAML()
}

// Handler returns an HTTP handler that serves the generated spec and Swagger UI.
func Handler(base string) http.Handler {
	h := swgui.New("Project Template API", base+"/openapi.json", base)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case base + "/openapi.json":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(specJSONBytes)
			return
		case base + "/openapi.yaml":
			w.Header().Set("Content-Type", "application/x-yaml")
			_, _ = w.Write(specYAMLBytes)
			return
		}
		h.ServeHTTP(w, r)
	})
}
