//go:build grpc

package cmd

import (
	"errors"
	"log"
	"net"
	"sync"

	"project-template/infrastructure/config"
	"project-template/pkg/lifecycle"
	"project-template/pkg/logger"
	ghandlers "project-template/server/grpc"

	grpcpkg "google.golang.org/grpc"
)

// Start configures dependencies and launches the gRPC server.
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

	server := GetGRPCServer(config.Cfg, loggerInstance)
	lifecycle.RegisterClose(GRPCCloseListener)
	lifecycle.RegisterShutdown(GRPCShutdownServer)

	if err = server.StartServer(); err != nil {
		if errors.Is(err, grpcpkg.ErrServerStopped) {
			loggerInstance.InfoF("server closed")
		} else {
			loggerInstance.InfoF("Failed to start the server: %v", err)
		}
	}
	if doneChan != nil {
		close(doneChan)
	}
}

// GRPCServer manages a gRPC listener and server instance.
type GRPCServer struct {
	server   *grpcpkg.Server
	listener net.Listener
	logger   logger.Logger
	cfg      *config.Config
}

var (
	grpcInstance     *GRPCServer
	grpcCreateOnce   sync.Once
	grpcShutdownOnce sync.Once
)

// GetGRPCServer ensures a singleton gRPC server instance.
func GetGRPCServer(cfg *config.Config, log logger.Logger) *GRPCServer {
	grpcCreateOnce.Do(func() {
		srv := ghandlers.NewServer(log)
		grpcInstance = &GRPCServer{
			server: srv,
			logger: log,
			cfg:    cfg,
		}
	})
	return grpcInstance
}

// StartServer listens and serves gRPC requests.
func (s *GRPCServer) StartServer() error {
	ln, err := net.Listen("tcp", s.cfg.GRPC.Host+":"+s.cfg.GRPC.Port)
	if err != nil {
		return err
	}

	s.listener = ln
	s.logger.InfoF("Starting gRPC server at " + ln.Addr().String())

	lifecycle.NotifyReady()
	return s.server.Serve(ln)
}

// GRPCCloseListener stops accepting new connections.
func GRPCCloseListener() {
	if grpcInstance == nil {
		return
	}
	if grpcInstance.listener != nil {
		_ = grpcInstance.listener.Close()
		grpcInstance.listener = nil
	}
}

// GRPCShutdownServer gracefully stops the gRPC server.
func GRPCShutdownServer() error {
	if grpcInstance == nil {
		return nil
	}
	var err error
	grpcShutdownOnce.Do(func() {
		grpcInstance.logger.InfoF("Shutting down the gRPC server")
		GRPCCloseListener()
		grpcInstance.server.GracefulStop()
	})
	return err
}
