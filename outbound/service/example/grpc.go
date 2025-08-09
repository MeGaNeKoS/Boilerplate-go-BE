package example

import (
	"context"
	"fmt"

	"project-template/infrastructure/config"
	"project-template/infrastructure/utils"
	"project-template/outbound/transport"
	"project-template/pb"
	"project-template/pkg/logger"
)

type exampleGRPCOutbound struct {
	log logger.Logger
}

func NewExampleGRPCOutbound(l logger.Logger) ExampleGRPCOutbound {
	return &exampleGRPCOutbound{log: l}
}

func (e *exampleGRPCOutbound) headers(ctx context.Context) map[string]string {
	tok, _ := utils.GetTokenCtx(ctx)
	return map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", string(tok)),
		"Parent-Id":     e.log.ParentID(),
	}
}

// getTransport returns a preconfigured gRPC outbound with host and metadata.
func (e *exampleGRPCOutbound) getTransport(ctx context.Context) *transport.GRPCOutbound {
	return &transport.GRPCOutbound{
		Host:     config.Cfg.Service.Example.Host,
		Metadata: e.headers(ctx),
	}
}

func (e *exampleGRPCOutbound) GetItem(ctx context.Context, id int) (*pb.Item, error) {
	call := e.getTransport(ctx).
		WithMethod("/pb.ItemService/GetItem").
		WithRequest(&pb.ItemID{Id: int32(id)}).
		WithResponse(&pb.Item{})
	resp, err := call.Invoke(ctx, e.log)
	if err != nil {
		return nil, fmt.Errorf("grpc call failed: %w", err)
	}
	item, ok := resp.(*pb.Item)
	if !ok {
		return nil, fmt.Errorf("unexpected response type %T", resp)
	}
	return item, nil
}
