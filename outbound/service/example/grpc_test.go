package example

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"project-template/infrastructure/config"
	"project-template/infrastructure/utils"
	"project-template/pb"
)

type itemSvc struct {
	pb.UnimplementedItemServiceServer
}

func (itemSvc) GetItem(ctx context.Context, in *pb.ItemID) (*pb.Item, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if md.Get("authorization")[0] != "Bearer tok" {
		return nil, grpc.Errorf(13, "missing auth")
	}
	return &pb.Item{Id: in.Id, Name: "n"}, nil
}

func TestGetItem(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := grpc.NewServer()
	pb.RegisterItemServiceServer(srv, itemSvc{})
	go srv.Serve(lis)
	defer srv.Stop()

	config.Cfg = &config.Config{Service: config.Service{Example: config.ServiceDetail{Host: lis.Addr().String()}}}
	ob := &exampleGRPCOutbound{log: stubLogger{parent: "p"}}
	ctx := utils.SetTokenCtx(context.Background(), utils.SecureString("tok"))
	item, err := ob.GetItem(ctx, 7)
	if err != nil {
		t.Fatalf("GetItem error: %v", err)
	}
	if item.GetId() != 7 || item.GetName() != "n" {
		t.Fatalf("bad item %#v", item)
	}
}
