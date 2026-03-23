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
	"project-template/server/rest"
	restmw "project-template/server/rest/middleware"
	"project-template/server/rest/routes"

	"project-template/infrastructure/utils"

	"github.com/MeGaNeKoS/neoma/adapters/neomachi/v5"
	"github.com/MeGaNeKoS/neoma/core"
	"github.com/MeGaNeKoS/neoma/middleware"
	"github.com/MeGaNeKoS/neoma/neoma"
	"github.com/coreos/go-systemd/v22/activation"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

// Start loads configuration, initializes dependencies and starts the HTTP server.
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
	if server == nil {
		loggerInstance.InfoF("Failed to initialize REST server")
		if doneChan != nil {
			close(doneChan)
		}
		return
	}
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
		if cfg.REST.Port != "" && cfg.REST.Port != "0" {
			if _, err := strconv.Atoi(cfg.REST.Port); err != nil {
				log.ErrorF("invalid REST port %q: %v", cfg.REST.Port, err)
				return
			}
		}

		base := strings.TrimRight(utils.NormalizeBasePath(cfg.Server.Endpoint.Based), "/")
		if base == "" {
			base = "/"
		}

		cfgInfo := neoma.DefaultConfig(cfg.AppName, cfg.Version)
		rest.Init(cfg.AppName)
		cfgInfo.ErrorHandler = rest.NewErrorHandler()
		if cfg.OpenAPI.Servers.Public != "" {
			cfgInfo.Servers = []*core.Server{{URL: strings.TrimSuffix(cfg.OpenAPI.Servers.Public, "/") + base}}
		}

		if cfg.OpenAPI.Version != "" {
			cfgInfo.OpenAPIVersion = cfg.OpenAPI.Version
		}
		if cfg.OpenAPI.Docs.Public.URL != "" {
			cfgInfo.Docs.Path = cfg.OpenAPI.Docs.Public.URL
		}
		if cfg.OpenAPI.Docs.Internal.URL != "" {
			cfgInfo.InternalSpec.Enabled = true
			cfgInfo.InternalSpec.DocsPath = cfg.OpenAPI.Docs.Internal.URL
			cfgInfo.InternalSpec.Path = cfg.OpenAPI.Docs.Internal.URL + "/openapi"
		}

		root := chi.NewRouter()
		root.Use(chimw.StripSlashes)

		// Mount neoma under the base path so spec/docs/schema routes
		// all live under the same prefix as the API operations.
		apiRouter := chi.NewRouter()
		api := neomachi.New(apiRouter, cfgInfo)
		grp := middleware.NewGroup(api)

		items := grp.Group("/items")
		items.UseDefaultTag("items")
		items.WithSecurity("bearerAuth", &core.SecurityScheme{
			Type:         "http",
			Scheme:       "bearer",
			BearerFormat: "JWT",
		}, restmw.WrapHTTPMiddleware(restmw.AuthMiddleware))
		routes.ItemsRouter(items)

		sys := grp.Group("/system")
		sys.UseDefaultTag("system")
		routes.SystemRouter(sys)

		files := grp.Group("/files")
		files.UseDefaultTag("files")
		routes.FilesRouter(files)

		root.Mount(base, apiRouter)

		log.InfoF("Registered routes:")
		if err := logRoutes(root, log); err != nil {
			log.ErrorF(fmt.Sprintf("%s", err))
			return
		}

		handler := restmw.ServiceMiddleware(restmw.ErrorFormatMiddleware(root))
		handler = restmw.LoggerMiddleware(handler)
		handler = restmw.RecoverMiddleware(log)(handler)

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

// RestCloseListener stops the server from accepting new connections.
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
		timeout := time.Duration(config.Cfg.Server.Timeout.Write) * time.Second
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		err = restInstance.httpServer.Shutdown(ctx)
	})
	return err
}
