//go:build !grpc && !kafka

package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"project-template/infrastructure/config"
	"project-template/pkg/lifecycle"
	"project-template/pkg/logger"
	"project-template/server/rest/handlers/resthuma"
	"project-template/server/rest/helpers"
	"project-template/server/rest/middleware"
	"project-template/server/rest/routes"

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
		router := chi.NewRouter()
		router.Use(chimw.StripSlashes)

		cfgInfo := huma.DefaultConfig("Project Template API", "0.1rc")
		if cfg != nil && cfg.Version != "" {
			cfgInfo.OpenAPI.Info.Version = cfg.Version
		}
		cfgInfo.OpenAPI.OpenAPI = "3.0.3"
		if cfg != nil && cfg.OpenAPI.Version != "" {
			cfgInfo.OpenAPI.OpenAPI = cfg.OpenAPI.Version
		}
		cfgInfo.OpenAPIPath = ""
		cfgInfo.DocsPath = ""

		api := humachi.New(router, cfgInfo)

		base := strings.TrimSuffix(utils.NormalizeBasePath(cfg.Server.Endpoint.Based), "/")
		grp := huma.NewGroup(api, base)

		items := huma.NewGroup(grp)
		routes.UseDefaultTag(items, "/items")
		items.UseMiddleware(middleware.HumaAuthMiddleware(items))
		routes.ItemsRouter(items)

		sys := huma.NewGroup(grp, "/system")
		routes.UseDefaultTag(sys, "/system")
		routes.SystemRouter(sys)

		files := huma.NewGroup(grp, "/files")
		routes.UseDefaultTag(files, "/files")
		routes.FilesRouter(files)

		resthuma.RegisterSchemas(api)
		resthuma.RegisterExamples(api)

		title := "API Reference"
		if cfgInfo.OpenAPI.Info != nil && cfgInfo.OpenAPI.Info.Title != "" {
			title = cfgInfo.OpenAPI.Info.Title
		}
		publicSpec := filterInternal(api.OpenAPI())
		internalSpec := stripInternalTag(api.OpenAPI())
		publicDocs := "/docs/public"
		internalDocs := "/docs/internal"
		if cfg.OpenAPI.Docs.Public != "" {
			publicDocs = cfg.OpenAPI.Docs.Public
		}
		if cfg.OpenAPI.Docs.Internal != "" {
			internalDocs = cfg.OpenAPI.Docs.Internal
		}
		registerDocs(api, publicSpec, "/openapi-public", publicDocs, title)
		registerDocs(api, internalSpec, "/openapi-internal", internalDocs, title)

		log.InfoF("Registered routes:")
		if err := logRoutes(router, log); err != nil {
			log.ErrorF(fmt.Sprintf("%s", err))
			return
		}

		handler := middleware.ServiceMiddleware(router)
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
				if t == helpers.InternalTag() {
					return nil
				}
			}
			opCopy := *op
			if params := helpers.InternalParamsFor(op.OperationID); len(params) > 0 {
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
			if t.Name != helpers.InternalTag() {
				tags = append(tags, t)
			}
		}
		filtered.Tags = tags
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
				if t != helpers.InternalTag() {
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
			if t.Name != helpers.InternalTag() {
				tags = append(tags, t)
			}
		}
		stripped.Tags = tags
	}
	return &stripped
}

func registerDocs(api huma.API, spec *huma.OpenAPI, openAPIPath, docsPath, title string) {
	var specYAML []byte
	api.Adapter().Handle(&huma.Operation{
		Method: http.MethodGet,
		Path:   openAPIPath + ".yaml",
	}, func(ctx huma.Context) {
		ctx.SetHeader("Content-Type", "application/vnd.oai.openapi+yaml")
		if specYAML == nil {
			specYAML, _ = spec.DowngradeYAML()
		}
		ctx.BodyWriter().Write(specYAML)
	})
	api.Adapter().Handle(&huma.Operation{
		Method: http.MethodGet,
		Path:   docsPath,
	}, func(ctx huma.Context) {
		ctx.SetHeader("Content-Type", "text/html")
		ctx.BodyWriter().Write([]byte(`<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="referrer" content="same-origin" />
    <meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no" />
    <title>` + title + `</title>
    <link href="https://unpkg.com/@stoplight/elements@9.0.0/styles.min.css" rel="stylesheet" />
    <script src="https://unpkg.com/@stoplight/elements@9.0.0/web-components.min.js" integrity="sha256-Tqvw1qE2abI+G6dPQBc5zbeHqfVwGoamETU3/TSpUw4=" crossorigin="anonymous"></script>
  </head>
  <body style="height: 100vh;">
    <elements-api apiDescriptionUrl="` + openAPIPath + `.yaml" router="hash" layout="sidebar" tryItCredentialsPolicy="same-origin" />
  </body>
</html>`))
	})
}
