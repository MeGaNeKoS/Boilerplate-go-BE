package transport

import (
	"context"
	"reflect"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"project-template/pkg/logger"
)

// GRPCOutbound contains information for making a generic gRPC call.
type GRPCOutbound struct {
	Host     string
	Method   string
	Request  interface{}
	Response interface{}
	Metadata map[string]string
}

// Invoke performs the gRPC call and unmarshals the response into the provided
// Response type. The returned value will be of the same concrete type as
// Response.
func (o *GRPCOutbound) Invoke(ctx context.Context, log logger.Logger) (interface{}, error) {
	// grpc.Dial is deprecated in favor of grpc.NewClient.
	// Use NewClient for establishing outbound connections.
	conn, err := grpc.NewClient(o.Host, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := conn.Close(); cerr != nil {
			log.WarnF("failed to close gRPC connection: %v", cerr)
		}
	}()

	resp := reflect.New(reflect.TypeOf(o.Response).Elem()).Interface()
	if len(o.Metadata) > 0 {
		md := metadata.New(o.Metadata)
		ctx = metadata.NewOutgoingContext(ctx, md)
	}
	if err := conn.Invoke(ctx, o.Method, o.Request, resp); err != nil {
		return nil, err
	}
	return resp, nil
}
