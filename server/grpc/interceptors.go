package grpc

import (
	"context"
	"runtime"

	"project-template/infrastructure/supervisor"
	"project-template/pkg/logger"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RecoverUnaryServerInterceptor converts panics into gRPC errors and logs the stack trace.
func RecoverUnaryServerInterceptor(log logger.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				stack := make([]byte, 8192)
				n := runtime.Stack(stack, false)
				log.ErrorF("Recovered panic in %s: %v\n%s", info.FullMethod, r, stack[:n])
				err = status.Error(codes.Internal, "internal server error")
				go func() {
					if rErr := supervisor.RequestRestart(); rErr != nil {
						log.ErrorF("restart request failed: %v", rErr)
					}
				}()
			}
		}()
		return handler(ctx, req)
	}
}

// LoggingUnaryServerInterceptor logs gRPC requests and their results.
func LoggingUnaryServerInterceptor(log logger.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		log.InfoF("gRPC call: %s", info.FullMethod)
		resp, err := handler(ctx, req)
		if err != nil {
			log.ErrorF("gRPC error: %v", err)
		}
		return resp, err
	}
}
