package grpc

import (
	"context"
	"strings"

	"project-template/infrastructure/config"
	"project-template/infrastructure/db"
	"project-template/infrastructure/dto"
	models "project-template/infrastructure/dto/item"
	"project-template/infrastructure/utils"
	"project-template/outbound"
	"project-template/pb"
	"project-template/pkg/logger"
	repo "project-template/repositories"
	services "project-template/services/item"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Server exposes the Item service over gRPC.
type Server struct {
	pb.UnimplementedItemServiceServer
	log logger.Logger
}

// NewServer creates a gRPC server with recovery and logging interceptors.
func NewServer(log logger.Logger) *grpc.Server {
	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			RecoverUnaryServerInterceptor(log),
			LoggingUnaryServerInterceptor(log),
		),
	)
	pb.RegisterItemServiceServer(srv, &Server{log: log})
	return srv
}

func getHeader(md metadata.MD, key string) string {
	if vals := md.Get(key); len(vals) > 0 {
		return vals[0]
	}
	return ""
}

func (s *Server) buildService(ctx context.Context) (context.Context, services.ServiceImpl, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	parentID := getHeader(md, "parent-id")
	if parentID == "" {
		id, _ := utils.UniqueIdByTime(86400)
		parentID = "-" + id
	}
	token := strings.TrimPrefix(getHeader(md, "authorization"), "Bearer ")

	reqLog := utils.GetLoggerFromContext(ctx)
	if reqLog == nil {
		childId, _ := utils.UniqueIdByTime(86400)
		reqLog, _ = logger.NewLogger(config.Cfg.LogTarget, parentID, childId)
	}

	var user dto.JWTUser
	if err := utils.GetJWTService().ParseJWT(token, &user); err != nil {
		reqLog.ErrorF("invalid token: %v", err)
		return ctx, nil, status.Error(codes.Unauthenticated, "invalid token")
	}

	ctx = utils.SetUserCtx(ctx, &user)

	ctx = utils.SetTokenCtx(ctx, utils.SecureString(token))
	repoImpl := repo.NewRepository(reqLog, db.GetDBInstance())
	outbounds := outbound.NewOutbound(reqLog)
	svc := services.NewService(repoImpl.GetItemRepository(), outbounds.Example().HTTP(), reqLog)
	return ctx, svc, nil
}

func (s *Server) CreateItem(ctx context.Context, in *pb.Item) (*pb.Item, error) {
	ctx, svc, err := s.buildService(ctx)
	if err != nil {
		return nil, err
	}
	item := models.Item{ID: int(in.Id), Name: in.Name}
	created, codeErr := svc.CreateItem(ctx, item)
	if codeErr != nil {
		return nil, status.Error(codeErr.GRPCCode, codeErr.Message)
	}
	return &pb.Item{Id: int32(created.ID), Name: created.Name}, nil
}

func (s *Server) ListItems(ctx context.Context, _ *pb.Empty) (*pb.ItemList, error) {
	ctx, svc, err := s.buildService(ctx)
	if err != nil {
		return nil, err
	}
	list, codeErr := svc.ListItems(ctx)
	if codeErr != nil {
		return nil, status.Error(codeErr.GRPCCode, codeErr.Message)
	}
	pbItems := make([]*pb.Item, len(list))
	for i, itm := range list {
		pbItems[i] = &pb.Item{Id: int32(itm.ID), Name: itm.Name}
	}
	return &pb.ItemList{Items: pbItems}, nil
}

func (s *Server) GetItem(ctx context.Context, in *pb.ItemID) (*pb.Item, error) {
	ctx, svc, err := s.buildService(ctx)
	if err != nil {
		return nil, err
	}
	itm, codeErr := svc.GetItem(ctx, int(in.Id))
	if codeErr != nil {
		return nil, status.Error(codeErr.GRPCCode, codeErr.Message)
	}
	return &pb.Item{Id: int32(itm.ID), Name: itm.Name}, nil
}

func (s *Server) UpdateItem(ctx context.Context, in *pb.Item) (*pb.Item, error) {
	ctx, svc, err := s.buildService(ctx)
	if err != nil {
		return nil, err
	}
	item := models.Item{ID: int(in.Id), Name: in.Name}
	updated, codeErr := svc.UpdateItem(ctx, item)
	if codeErr != nil {
		return nil, status.Error(codeErr.GRPCCode, codeErr.Message)
	}
	return &pb.Item{Id: int32(updated.ID), Name: updated.Name}, nil
}

func (s *Server) DeleteItem(ctx context.Context, in *pb.ItemID) (*pb.Empty, error) {
	ctx, svc, err := s.buildService(ctx)
	if err != nil {
		return nil, err
	}
	if codeErr := svc.DeleteItem(ctx, int(in.Id)); codeErr != nil {
		return nil, status.Error(codeErr.GRPCCode, codeErr.Message)
	}
	return &pb.Empty{}, nil
}
