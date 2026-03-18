//go:build !grpc && !kafka

package cmd

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"project-template/infrastructure/config"
	"project-template/pkg/lifecycle"
	"project-template/pkg/logger"
	"project-template/server/rest/middleware"
	"project-template/server/rest/routes"
	restutils "project-template/server/rest/utils"

	"project-template/infrastructure/utils"

	"github.com/coreos/go-systemd/v22/activation"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

// Start loads configuration, initializes dependencies and starts the HTTP server.
// If doneChan is provided, it will be closed when the function exits due to a version flag.
func Start(doneChan chan struct{}) {
	filePath, exit := parseFlags(doneChan)
	if exit {
		return
	}

	loggerInstance, err := initDependencies(filePath)
	if err != nil {
		log.Printf("Initialization failed: %v", err)
		if doneChan != nil {
			close(doneChan)
		}
		return
	}
	defer logger.CloseLogFile()

	server := GetRESTServer(config.Cfg, loggerInstance)
	lifecycle.RegisterClose(RestCloseListener)
	lifecycle.RegisterShutdown(RestShutdownServer)

	if err = server.StartServer(); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			loggerInstance.InfoF("server closed")
		} else {
			loggerInstance.InfoF("Failed to start the server: %v", err)
		}
	}
	if doneChan != nil {
		close(doneChan)
	}
}

var chiWalk = chi.Walk
var osWriteFile = os.WriteFile
var jsonMarshal = json.Marshal

func logRoutes(r chi.Routes, log logger.Logger) error {
	routeMap := map[string][]string{}
	err := chiWalk(r, func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		routeMap[route] = append(routeMap[route], method)
		return nil
	})
	if err != nil {
		return err
	}

	registeredRoutes := make([]string, 0, len(routeMap))
	for rt := range routeMap {
		registeredRoutes = append(registeredRoutes, rt)
	}
	sort.Strings(registeredRoutes)

	for _, rt := range registeredRoutes {
		ms := routeMap[rt]
		sort.Strings(ms)
		log.InfoF("%s %s", strings.Join(ms, ","), rt)
	}

	return nil
}

type RestServer struct {
	httpServer *http.Server
	listener   net.Listener
	logger     logger.Logger
}

var (
	restInstance     *RestServer
	restCreateOnce   sync.Once
	restShutdownOnce sync.Once
)

// GetRESTServer ensures a singleton REST server instance.
func GetRESTServer(cfg *config.Config, log logger.Logger) *RestServer {
	restCreateOnce.Do(func() {
		if cfg != nil {
			restutils.SetPrivateCIDRs(cfg.Server.InternalCIDRs)
		} else {
			restutils.SetPrivateCIDRs(nil)
		}
		router := chi.NewRouter()
		router.Use(chimw.StripSlashes)

		cfgInfo := huma.DefaultConfig("Project Template API", "0.1rc")
		cfgInfo.CreateHooks = nil
		cfgInfo.Transformers = append(cfgInfo.Transformers, restutils.SchemaLinkTransformer)
		if cfg != nil && cfg.Version != "" {
			cfgInfo.Info.Version = cfg.Version
		}
		cfgInfo.OpenAPI.OpenAPI = "3.0.3"
		if cfg != nil && cfg.OpenAPI.Version != "" {
			cfgInfo.OpenAPI.OpenAPI = cfg.OpenAPI.Version
		}
		publicDocs := "/docs/public"
		publicSchemaPart := "/schema/v1"
		internalDocs := "/docs/internal"
		internalSchemaPart := "/schema"
		if cfg != nil {
			if cfg.OpenAPI.Docs.Public.URL != "" {
				publicDocs = cfg.OpenAPI.Docs.Public.URL
			}
			if cfg.OpenAPI.Docs.Public.Schema != "" {
				publicSchemaPart = cfg.OpenAPI.Docs.Public.Schema
			}
			if cfg.OpenAPI.Docs.Internal.URL != "" {
				internalDocs = cfg.OpenAPI.Docs.Internal.URL
			}
			if cfg.OpenAPI.Docs.Internal.Schema != "" {
				internalSchemaPart = cfg.OpenAPI.Docs.Internal.Schema
			}
		}
		publicSchemaPath := publicDocs + publicSchemaPart
		internalSchemaPath := internalDocs + internalSchemaPart
		cfgInfo.SchemasPath = internalSchemaPath
		cfgInfo.OpenAPIPath = ""
		cfgInfo.DocsPath = ""

		base := strings.TrimSuffix(utils.NormalizeBasePath(cfg.Server.Endpoint.Based), "/")
		listenAddr := fmt.Sprintf("//%s:%s%s", cfg.REST.Host, cfg.REST.Port, base)
		publicServerURL := listenAddr
		if cfg.OpenAPI.Servers.Public != "" {
			publicServerURL = strings.TrimSuffix(cfg.OpenAPI.Servers.Public, "/") + base
		}
		internalServerURL := publicServerURL
		if cfg.OpenAPI.Servers.Internal != "" {
			internalServerURL = strings.TrimSuffix(cfg.OpenAPI.Servers.Internal, "/") + base
		}
		cfgInfo.Servers = []*huma.Server{{URL: publicServerURL}}
		api := humachi.New(router, cfgInfo)

		grp := huma.NewGroup(api, base)

		items := routes.NewGroup(grp, "/items")
		items.UseMiddleware(middleware.HumaAuthMiddleware(items))
		routes.ItemsRouter(items)

		sys := routes.NewGroup(grp, "/system")
		routes.SystemRouter(sys)

		files := routes.NewGroup(grp, "/files")
		routes.FilesRouter(files)

		restutils.RegisterSchemas(api)
		restutils.RegisterExamples(api)

		title := "API Reference"
		if cfgInfo.Info != nil && cfgInfo.Info.Title != "" {
			title = cfgInfo.Info.Title
		}
		setupDocsAndSchemas(api, router, cfg, base, publicServerURL, internalServerURL, publicDocs, publicSchemaPath, internalDocs, internalSchemaPath, title)

		log.InfoF("Registered routes:")
		if err := logRoutes(router, log); err != nil {
			log.ErrorF(fmt.Sprintf("%s", err))
			return
		}

		handler := middleware.ServiceMiddleware(middleware.ErrorFormatMiddleware(router))
		handler = middleware.LoggerMiddleware(handler)
		handler = middleware.RecoverMiddleware(log)(handler)

		httpSrv := &http.Server{
			Addr:         cfg.REST.Host + ":" + cfg.REST.Port,
			Handler:      handler,
			ReadTimeout:  time.Duration(cfg.Server.Timeout.Read) * time.Second,
			WriteTimeout: time.Duration(cfg.Server.Timeout.Write) * time.Second,
			IdleTimeout:  time.Duration(cfg.Server.Timeout.Idle) * time.Second,
		}

		restInstance = &RestServer{
			httpServer: httpSrv,
			logger:     log,
		}
	})
	return restInstance
}

// StartServer begins listening for HTTP requests and blocks until the server stops.
func (s *RestServer) StartServer() error {
	if s.httpServer == nil {
		return errors.New("http server is not initialized")
	}
	var ln net.Listener
	var err error
	useSocket := false
	if envVal, exists := os.LookupEnv("SYSTEMD_SOCKET_ACTIVATION"); exists {
		parsed, pErr := strconv.ParseBool(envVal)
		if pErr != nil {
			return fmt.Errorf("invalid value for SYSTEMD_SOCKET_ACTIVATION: %w", pErr)
		}
		useSocket = parsed
	}
	if useSocket {
		listeners, lErr := activation.Listeners()
		if lErr != nil {
			return lErr
		}
		if len(listeners) == 0 {
			return errors.New("no systemd listeners found")
		}
		ln = listeners[0]
	} else {
		ln, err = net.Listen("tcp", s.httpServer.Addr)
		if err != nil {
			return err
		}
	}
	s.listener = ln
	s.logger.InfoF("Starting server at " + ln.Addr().String())

	lifecycle.NotifyReady()
	return s.httpServer.Serve(ln)
}

// RestCloseListener stops the server from accepting new connections. It is safe to call multiple times.
func RestCloseListener() {
	if restInstance == nil {
		return
	}
	if restInstance.listener != nil {
		_ = restInstance.listener.Close()
		restInstance.listener = nil
	}
}

// RestShutdownServer gracefully shuts down the HTTP server.
func RestShutdownServer() error {
	if restInstance == nil {
		return nil
	}
	var err error
	restShutdownOnce.Do(func() {
		restInstance.logger.InfoF("Shutting down the server (via panic or signal)")
		// Derive the shutdown timeout from configuration.
		timeout := time.Duration(config.Cfg.Server.Timeout.Write) * time.Second
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		err = restInstance.httpServer.Shutdown(ctx)
	})
	return err
}

func filterInternal(o *huma.OpenAPI) *huma.OpenAPI {
	filtered := *o
	filtered.Paths = map[string]*huma.PathItem{}

	for path, item := range o.Paths {
		copyItem := &huma.PathItem{}
		removeInternal := func(op *huma.Operation) *huma.Operation {
			if op == nil {
				return nil
			}
			for _, t := range op.Tags {
				if t == restutils.InternalTag() {
					return nil
				}
			}
			opCopy := *op
			if params := restutils.InternalParamsFor(op.OperationID); len(params) > 0 {
				keep := make([]*huma.Param, 0, len(op.Parameters))
				for _, p := range op.Parameters {
					drop := false
					for _, ip := range params {
						if p.Name == ip.Name && p.In == ip.In {
							drop = true
							break
						}
					}
					if !drop {
						keep = append(keep, p)
					}
				}
				opCopy.Parameters = keep
			}
			return &opCopy
		}
		copyItem.Get = removeInternal(item.Get)
		copyItem.Put = removeInternal(item.Put)
		copyItem.Post = removeInternal(item.Post)
		copyItem.Delete = removeInternal(item.Delete)
		copyItem.Options = removeInternal(item.Options)
		copyItem.Head = removeInternal(item.Head)
		copyItem.Patch = removeInternal(item.Patch)
		copyItem.Trace = removeInternal(item.Trace)
		if copyItem.Get != nil || copyItem.Put != nil || copyItem.Post != nil || copyItem.Delete != nil || copyItem.Options != nil || copyItem.Head != nil || copyItem.Patch != nil || copyItem.Trace != nil {
			filtered.Paths[path] = copyItem
		}
	}

	if len(o.Tags) > 0 {
		tags := make([]*huma.Tag, 0, len(o.Tags))
		for _, t := range o.Tags {
			if t.Name != restutils.InternalTag() {
				tags = append(tags, t)
			}
		}
		filtered.Tags = tags
	}

	if o.Components != nil && o.Components.Schemas != nil {
		reg := huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)
		for name, s := range o.Components.Schemas.Map() {
			reg.Map()[name] = s
		}
		filtered.Components = &huma.Components{Schemas: reg}
		refs := map[string]struct{}{}
		seen := map[*huma.Schema]bool{}
		for _, item := range filtered.Paths {
			collectOpRefs(item.Get, reg, refs, seen)
			collectOpRefs(item.Put, reg, refs, seen)
			collectOpRefs(item.Post, reg, refs, seen)
			collectOpRefs(item.Delete, reg, refs, seen)
			collectOpRefs(item.Options, reg, refs, seen)
			collectOpRefs(item.Head, reg, refs, seen)
			collectOpRefs(item.Patch, reg, refs, seen)
			collectOpRefs(item.Trace, reg, refs, seen)
		}
		m := reg.Map()
		for name := range m {
			if _, ok := refs[name]; !ok {
				delete(m, name)
			}
		}
	}

	return &filtered
}

func stripInternalTag(o *huma.OpenAPI) *huma.OpenAPI {
	stripped := *o
	stripped.Paths = map[string]*huma.PathItem{}
	for path, item := range o.Paths {
		copyItem := &huma.PathItem{}
		removeTag := func(op *huma.Operation) *huma.Operation {
			if op == nil {
				return nil
			}
			opCopy := *op
			tags := make([]string, 0, len(op.Tags))
			for _, t := range op.Tags {
				if t != restutils.InternalTag() {
					tags = append(tags, t)
				}
			}
			opCopy.Tags = tags
			return &opCopy
		}
		copyItem.Get = removeTag(item.Get)
		copyItem.Put = removeTag(item.Put)
		copyItem.Post = removeTag(item.Post)
		copyItem.Delete = removeTag(item.Delete)
		copyItem.Options = removeTag(item.Options)
		copyItem.Head = removeTag(item.Head)
		copyItem.Patch = removeTag(item.Patch)
		copyItem.Trace = removeTag(item.Trace)
		stripped.Paths[path] = copyItem
	}
	if len(o.Tags) > 0 {
		tags := make([]*huma.Tag, 0, len(o.Tags))
		for _, t := range o.Tags {
			if t.Name != restutils.InternalTag() {
				tags = append(tags, t)
			}
		}
		stripped.Tags = tags
	}
	return &stripped
}

func collectOpRefs(op *huma.Operation, reg huma.Registry, refs map[string]struct{}, seen map[*huma.Schema]bool) {
	if op == nil {
		return
	}
	for _, p := range op.Parameters {
		collectSchemaRefs(p.Schema, reg, refs, seen)
	}
	if op.RequestBody != nil {
		for _, mt := range op.RequestBody.Content {
			collectSchemaRefs(mt.Schema, reg, refs, seen)
		}
	}
	for _, r := range op.Responses {
		for _, mt := range r.Content {
			collectSchemaRefs(mt.Schema, reg, refs, seen)
		}
		for _, h := range r.Headers {
			collectSchemaRefs(h.Schema, reg, refs, seen)
		}
	}
}

func collectSchemaRefs(s *huma.Schema, reg huma.Registry, refs map[string]struct{}, seen map[*huma.Schema]bool) {
	if s == nil {
		return
	}
	if s.Ref != "" {
		if strings.HasPrefix(s.Ref, "#/components/schemas/") {
			name := strings.TrimPrefix(s.Ref, "#/components/schemas/")
			if _, ok := refs[name]; !ok {
				refs[name] = struct{}{}
				collectSchemaRefs(reg.SchemaFromRef(s.Ref), reg, refs, seen)
			}
		}
		return
	}
	if seen[s] {
		return
	}
	seen[s] = true
	for _, p := range s.Properties {
		collectSchemaRefs(p, reg, refs, seen)
	}
	collectSchemaRefs(s.Items, reg, refs, seen)
	for _, a := range s.AllOf {
		collectSchemaRefs(a, reg, refs, seen)
	}
	for _, a := range s.OneOf {
		collectSchemaRefs(a, reg, refs, seen)
	}
	for _, a := range s.AnyOf {
		collectSchemaRefs(a, reg, refs, seen)
	}
	if ap, ok := s.AdditionalProperties.(*huma.Schema); ok {
		collectSchemaRefs(ap, reg, refs, seen)
	}
}

func stripBasePath(o *huma.OpenAPI, base string) {
	base = strings.TrimSuffix(base, "/")
	if base == "" {
		return
	}
	newPaths := map[string]*huma.PathItem{}
	for p, item := range o.Paths {
		if strings.HasPrefix(p, base) {
			np := strings.TrimPrefix(p, base)
			if np == "" {
				np = "/"
			}
			newPaths[np] = item
		} else {
			newPaths[p] = item
		}
	}
	o.Paths = newPaths
}

func registerSchemaRoute(r chi.Router, routePath, fsPath, outputDir string) {
	base := strings.TrimSuffix(routePath, "/")
	fsBase := filepath.Join(outputDir, strings.TrimPrefix(strings.TrimSuffix(fsPath, "/"), "/"))
	r.Get(base+"/{schema}", func(w http.ResponseWriter, req *http.Request) {
		name := chi.URLParam(req, "schema")
		if !strings.HasSuffix(name, ".json") {
			name += ".json"
		}
		fp := filepath.Join(fsBase, name)
		data, err := os.ReadFile(fp)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		sum := sha256.Sum256(data)
		etag := fmt.Sprintf("\"%x\"", sum)
		if match := req.Header.Get("If-None-Match"); match != "" && match == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("Content-Type", "application/schema+json")
		w.Header().Set("Cache-Control", "public, max-age=300")
		w.Header().Set("ETag", etag)
		_, _ = w.Write(data)
	})
}

func setupDocsAndSchemas(api huma.API, router chi.Router, cfg *config.Config, base, publicServerURL, internalServerURL, publicDocs, publicSchemaPath, internalDocs, internalSchemaPath, title string) {
	publicSpec := filterInternal(api.OpenAPI())
	internalSpec := stripInternalTag(api.OpenAPI())
	stripBasePath(publicSpec, base)
	stripBasePath(internalSpec, base)
	if len(publicSpec.Servers) > 0 {
		publicSpec.Servers[0].URL = publicServerURL
	}
	if len(internalSpec.Servers) > 0 {
		internalSpec.Servers[0].URL = internalServerURL
	}
	restutils.RewriteSchemaExamples(publicSpec, publicServerURL+publicSchemaPath)
	restutils.RewriteSchemaExamples(internalSpec, internalServerURL+internalSchemaPath)
	restutils.RewriteSchemaLinks(publicSpec, false)
	restutils.RewriteSchemaLinks(internalSpec, true)
	restutils.RewriteExampleNames(publicSpec)
	restutils.RewriteExampleNames(internalSpec)
	docsServerURL := strings.TrimSuffix(publicServerURL, base)
	internalDocsServerURL := strings.TrimSuffix(internalServerURL, base)
	outputDir := os.TempDir()
	if cfg != nil && cfg.OpenAPI.OutputDir != "" {
		outputDir = cfg.OpenAPI.OutputDir
	}
	_ = ensureDocs(publicSpec, docsServerURL, publicDocs, publicSchemaPath, outputDir)
	_ = ensureDocs(internalSpec, internalDocsServerURL, internalDocs, internalSchemaPath, outputDir)
	renderer := "stoplight"
	if cfg != nil && cfg.OpenAPI.Docs.Renderer != "" {
		renderer = cfg.OpenAPI.Docs.Renderer
	}
	registerDocs(api, publicSpec, "/openapi-public", publicDocs, title, renderer)
	registerDocs(api, internalSpec, "/openapi-internal", internalDocs, title, renderer)
	registerSchemaRoute(router, publicSchemaPath, publicSchemaPath, outputDir)
	registerSchemaRoute(router, internalSchemaPath, internalSchemaPath, outputDir)
}

func ensureDocs(spec *huma.OpenAPI, serverURL, docsPath, schemaPath, outputDir string) error {
	docDir := filepath.Join(outputDir, strings.TrimPrefix(docsPath, "/"))
	schemaDir := filepath.Join(outputDir, strings.TrimPrefix(schemaPath, "/"))
	if spec.Components != nil && spec.Components.Schemas != nil {
		for _, s := range spec.Components.Schemas.Map() {
			stripAllOfAdditionalProperties(s)
		}
	}
	if _, err := os.Stat(filepath.Join(docDir, "openapi.json")); os.IsNotExist(err) {
		if err = os.MkdirAll(docDir, 0o755); err != nil {
			return err
		}
		b, err := jsonMarshal(spec)
		if err != nil {
			return err
		}
		if err = osWriteFile(filepath.Join(docDir, "openapi.json"), b, 0o644); err != nil {
			return err
		}
	}
	// Generate schema files if the directory is missing or contains no
	// non-hidden entries (e.g. only a .gitkeep file).
	entries, err := os.ReadDir(schemaDir)
	count := 0
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		count++
	}
	if os.IsNotExist(err) || count == 0 {
		if err = os.MkdirAll(schemaDir, 0o755); err != nil {
			return err
		}
		reg := spec.Components.Schemas
		for name, s := range reg.Map() {
			flat := flattenSchema(reg, s, map[*huma.Schema]bool{})
			stripAllOfAdditionalProperties(flat)
			data, err := json.Marshal(flat)
			if err != nil {
				return err
			}
			var m map[string]any
			if err = json.Unmarshal(data, &m); err != nil {
				return err
			}
			schemaURL := "https://spec.openapis.org/oas/3.0/schema/2021-09-28#/$defs/Schema"
			if strings.HasPrefix(spec.OpenAPI, "3.1") {
				schemaURL = "https://spec.openapis.org/oas/3.1/schema/2022-02-27#/$defs/Schema"
			}
			m["$schema"] = schemaURL
			m["$id"] = fmt.Sprintf("%s%s/%s.json", serverURL, schemaPath, name)
			out, err := json.MarshalIndent(m, "", "  ")
			if err != nil {
				return err
			}
			if err = osWriteFile(filepath.Join(schemaDir, name+".json"), out, 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}

func flattenSchema(reg huma.Registry, s *huma.Schema, seen map[*huma.Schema]bool) *huma.Schema {
	if s == nil {
		return nil
	}
	if s.Ref != "" {
		return flattenSchema(reg, reg.SchemaFromRef(s.Ref), seen)
	}
	if seen[s] {
		return &huma.Schema{}
	}
	seen[s] = true

	out := *s
	out.Ref = ""
	if len(s.Properties) > 0 {
		out.Properties = make(map[string]*huma.Schema, len(s.Properties))
		for k, v := range s.Properties {
			out.Properties[k] = flattenSchema(reg, v, seen)
		}
	}
	if s.Items != nil {
		out.Items = flattenSchema(reg, s.Items, seen)
	}
	switch ap := s.AdditionalProperties.(type) {
	case *huma.Schema:
		out.AdditionalProperties = flattenSchema(reg, ap, seen)
	default:
		out.AdditionalProperties = ap
	}
	if len(s.AllOf) > 0 {
		out.AllOf = make([]*huma.Schema, len(s.AllOf))
		for i, v := range s.AllOf {
			out.AllOf[i] = flattenSchema(reg, v, seen)
		}
	}
	if len(s.AnyOf) > 0 {
		out.AnyOf = make([]*huma.Schema, len(s.AnyOf))
		for i, v := range s.AnyOf {
			out.AnyOf[i] = flattenSchema(reg, v, seen)
		}
	}
	if len(s.OneOf) > 0 {
		out.OneOf = make([]*huma.Schema, len(s.OneOf))
		for i, v := range s.OneOf {
			out.OneOf[i] = flattenSchema(reg, v, seen)
		}
	}
	if s.Not != nil {
		out.Not = flattenSchema(reg, s.Not, seen)
	}
	return &out
}

func stripAllOfAdditionalProperties(s *huma.Schema) {
	if s == nil {
		return
	}

	if ap, ok := s.AdditionalProperties.(*huma.Schema); ok {
		stripAllOfAdditionalProperties(ap)
	}
	s.AdditionalProperties = nil

	if s.Items != nil {
		stripAllOfAdditionalProperties(s.Items)
	}
	for _, sub := range s.Properties {
		stripAllOfAdditionalProperties(sub)
	}
	for _, sub := range s.AllOf {
		stripAllOfAdditionalProperties(sub)
	}
	for _, sub := range s.AnyOf {
		stripAllOfAdditionalProperties(sub)
	}
	for _, sub := range s.OneOf {
		stripAllOfAdditionalProperties(sub)
	}
	if s.Not != nil {
		stripAllOfAdditionalProperties(s.Not)
	}
}

func registerDocs(api huma.API, spec *huma.OpenAPI, openAPIPath, docsPath, title, renderer string) {
	var specYAML, specJSON []byte
	api.Adapter().Handle(&huma.Operation{
		Method: http.MethodGet,
		Path:   openAPIPath + ".yaml",
	}, func(ctx huma.Context) {
		ctx.SetHeader("Content-Type", "application/vnd.oai.openapi+yaml")
		if specYAML == nil {
			var err error
			if strings.HasPrefix(spec.OpenAPI, "3.1") {
				specYAML, err = spec.YAML()
			} else {
				specYAML, err = spec.DowngradeYAML()
			}
			if err != nil {
				_, _ = ctx.BodyWriter().Write([]byte(err.Error()))
				return
			}
		}
		_, _ = ctx.BodyWriter().Write(specYAML)
	})

	api.Adapter().Handle(&huma.Operation{
		Method: http.MethodGet,
		Path:   docsPath + "/openapi.json",
	}, func(ctx huma.Context) {
		ctx.SetHeader("Content-Type", "application/json")
		if specJSON == nil {
			var err error
			specJSON, err = json.Marshal(spec)
			if err != nil {
				_, _ = ctx.BodyWriter().Write([]byte(err.Error()))
				return
			}
		}
		_, _ = ctx.BodyWriter().Write(specJSON)
	})

	api.Adapter().Handle(&huma.Operation{
		Method: http.MethodGet,
		Path:   docsPath,
	}, func(ctx huma.Context) {
		ctx.SetHeader("Content-Type", "text/html")

		page := `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="referrer" content="same-origin" />
    <meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no" />
    <title>` + title + `</title>
`

		switch strings.ToLower(renderer) {
		case "swagger", "swaggerui", "swagger-ui":
			page += `    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist/swagger-ui.css" />
  </head>
  <body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist/swagger-ui-standalone-preset.js"></script>
    <script>
      window.onload = () => {
        SwaggerUIBundle({
          url: '` + openAPIPath + `.yaml',
          dom_id: '#swagger-ui',
          presets: [SwaggerUIBundle.presets.apis, SwaggerUIStandalonePreset],
          layout: 'BaseLayout'
        });
      };
    </script>
  </body>
</html>`
		default:
			page += `    <link href="https://unpkg.com/@stoplight/elements/styles.min.css" rel="stylesheet" />
    <script src="https://unpkg.com/@stoplight/elements/web-components.min.js" crossorigin="anonymous"></script>
  </head>
  <body style="height: 100vh;">
    <elements-api apiDescriptionUrl="` + openAPIPath + `.yaml" router="hash" layout="sidebar" tryItCredentialsPolicy="same-origin" />
  </body>
</html>`
		}

		_, _ = ctx.BodyWriter().Write([]byte(page))
	})
}
