package grpc

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"project-template/infrastructure/config"
	"project-template/infrastructure/db"
	models "project-template/infrastructure/dto/item"
	"project-template/infrastructure/utils"
	"project-template/outbound"
	"project-template/outbound/service/example"
	"project-template/pb"
	"project-template/pkg/code"
	"project-template/pkg/logger"
	repo "project-template/repositories"
	repoitem "project-template/repositories/item"

	"github.com/bouk/monkey"
)

func TestGetHeader(t *testing.T) {
	md := metadata.Pairs("a", "b")
	if getHeader(md, "a") != "b" {
		t.Fatalf("wrong header")
	}
	if getHeader(md, "missing") != "" {
		t.Fatalf("expected empty")
	}
}

func TestNewServerRegistersService(t *testing.T) {
	srv := NewServer(stubLogger{})
	info := srv.GetServiceInfo()
	if _, ok := info["pb.ItemService"]; !ok {
		t.Fatalf("ItemService not registered")
	}
}

// --- stubs and helpers ---
type stubLogger struct{}

func (stubLogger) DebugF(string, ...interface{}) {}
func (stubLogger) InfoF(string, ...interface{})  {}
func (stubLogger) WarnF(string, ...interface{})  {}
func (stubLogger) ErrorF(string, ...interface{}) {}
func (stubLogger) FatalF(string, ...interface{}) {}
func (stubLogger) ParentID() string              { return "" }
func (stubLogger) ChildID() string               { return "" }
func (stubLogger) CloseLogFile()                 {}

type dummyItemRepo struct{}

func (dummyItemRepo) Create(context.Context, *models.Item) error { return nil }
func (dummyItemRepo) List(context.Context) ([]models.Item, error) {
	return []models.Item{{ID: 1, Name: "a"}}, nil
}
func (dummyItemRepo) Get(_ context.Context, id int) (*models.Item, error) {
	return &models.Item{ID: id, Name: "b"}, nil
}
func (dummyItemRepo) Update(context.Context, *models.Item) error { return nil }
func (dummyItemRepo) Delete(context.Context, int) error          { return nil }

type dummyItemRepoErr struct{}

func (dummyItemRepoErr) Create(context.Context, *models.Item) error { return errors.New("bad create") }
func (dummyItemRepoErr) List(context.Context) ([]models.Item, error) {
	return nil, errors.New("bad list")
}
func (dummyItemRepoErr) Get(context.Context, int) (*models.Item, error) {
	return nil, code.ErrItemNotFound
}
func (dummyItemRepoErr) Update(context.Context, *models.Item) error { return errors.New("bad update") }
func (dummyItemRepoErr) Delete(context.Context, int) error          { return errors.New("bad delete") }

type stubExampleOutbound struct{}

func (stubExampleOutbound) FetchItemByID(_ context.Context, id int) (models.Item, error) {
	return models.Item{ID: id}, nil
}
func (stubExampleOutbound) FetchItemByFilter(_ context.Context, filter string) ([]models.Item, error) {
	return nil, nil
}

type stubExampleAgg struct{}

func (stubExampleAgg) HTTP() example.ExampleOutbound       { return stubExampleOutbound{} }
func (stubExampleAgg) GRPC() example.ExampleGRPCOutbound   { return nil }
func (stubExampleAgg) Kafka() example.ExampleKafkaOutbound { return nil }

type stubOutbound struct{}

func (stubOutbound) Example() example.Service { return stubExampleAgg{} }

type stubService struct{}

func (stubService) CreateItem(_ context.Context, item models.Item) (models.Item, *code.Code) {
	return item, nil
}
func (stubService) ListItems(_ context.Context) ([]models.Item, *code.Code) { return nil, nil }
func (stubService) GetItem(_ context.Context, id int) (models.Item, *code.Code) {
	return models.Item{}, nil
}
func (stubService) UpdateItem(_ context.Context, item models.Item) (models.Item, *code.Code) {
	return models.Item{}, nil
}
func (stubService) DeleteItem(_ context.Context, id int) *code.Code { return nil }

type stubService2 struct{}

func (stubService2) CreateItem(_ context.Context, item models.Item) (models.Item, *code.Code) {
	return item, nil
}
func (stubService2) ListItems(_ context.Context) ([]models.Item, *code.Code) {
	return []models.Item{{ID: 1, Name: "a"}}, nil
}
func (stubService2) GetItem(_ context.Context, id int) (models.Item, *code.Code) {
	return models.Item{ID: id, Name: "b"}, nil
}
func (stubService2) UpdateItem(_ context.Context, item models.Item) (models.Item, *code.Code) {
	return item, nil
}
func (stubService2) DeleteItem(_ context.Context, id int) *code.Code { return nil }

type stubServiceErr struct{}

func (stubServiceErr) CreateItem(_ context.Context, item models.Item) (models.Item, *code.Code) {
	return models.Item{}, code.ErrInternalServerError
}
func (stubServiceErr) ListItems(_ context.Context) ([]models.Item, *code.Code) {
	return nil, code.ErrInternalServerError
}
func (stubServiceErr) GetItem(_ context.Context, id int) (models.Item, *code.Code) {
	return models.Item{}, code.ErrItemNotFound
}
func (stubServiceErr) UpdateItem(_ context.Context, item models.Item) (models.Item, *code.Code) {
	return models.Item{}, code.ErrInternalServerError
}
func (stubServiceErr) DeleteItem(_ context.Context, id int) *code.Code {
	return code.ErrInternalServerError
}

func writeKeyFiles(t *testing.T, dir string) (string, string) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	privPath := filepath.Join(dir, "priv.pem")
	pubPath := filepath.Join(dir, "pub.pem")

	privBytes := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	if err := os.WriteFile(privPath, privBytes, 0600); err != nil {
		t.Fatal(err)
	}
	pubDer, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	pubBytes := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDer})
	if err := os.WriteFile(pubPath, pubBytes, 0600); err != nil {
		t.Fatal(err)
	}
	return privPath, pubPath
}

func TestCreateItemUnauthenticated(t *testing.T) {
	defer monkey.UnpatchAll()
	patchRepo := monkey.Patch(repo.NewRepository, func(logger.Logger, db.DB) repo.Impl { return &repo.Repository{} })
	t.Cleanup(patchRepo.Unpatch)
	monkey.PatchInstanceMethod(reflect.TypeOf(&repo.Repository{}), "GetItemRepository", func(*repo.Repository) repoitem.Repository { return dummyItemRepo{} })
	monkey.Patch(outbound.NewOutbound, func(logger.Logger) outbound.Impl { return stubOutbound{} })

	dir := t.TempDir()
	priv, pub := writeKeyFiles(t, dir)
	config.Cfg = &config.Config{AppName: "APP", LogTarget: config.LogConfig{Path: dir, FileName: "app.log"}, Server: config.ServerConfig{JWT: config.JWTConfig{PrivateKey: priv, PublicKey: pub}}}

	if err := utils.InitializeJWTService(true, true); err != nil {
		t.Fatal(err)
	}
	srv := &Server{log: stubLogger{}}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "bad"))
	ctx = utils.SetLoggerToContext(ctx, stubLogger{})
	_, err := srv.CreateItem(ctx, &pb.Item{Id: 1, Name: "x"})
	st, _ := status.FromError(err)
	if st.Code() != codes.Unauthenticated {
		t.Fatalf("expected unauthenticated got %v", st.Code())
	}
}

func TestCreateItemSuccess(t *testing.T) {
	defer monkey.UnpatchAll()
	patchRepo := monkey.Patch(repo.NewRepository, func(logger.Logger, db.DB) repo.Impl { return &repo.Repository{} })
	t.Cleanup(patchRepo.Unpatch)
	monkey.PatchInstanceMethod(reflect.TypeOf(&repo.Repository{}), "GetItemRepository", func(*repo.Repository) repoitem.Repository { return dummyItemRepo{} })
	monkey.Patch(outbound.NewOutbound, func(logger.Logger) outbound.Impl { return stubOutbound{} })

	dir := t.TempDir()
	priv, pub := writeKeyFiles(t, dir)
	config.Cfg = &config.Config{AppName: "APP", LogTarget: config.LogConfig{Path: dir, FileName: "app.log"}, Server: config.ServerConfig{JWT: config.JWTConfig{PrivateKey: priv, PublicKey: pub}}}

	if err := utils.InitializeJWTService(true, true); err != nil {
		t.Fatal(err)
	}
	tok, _ := utils.GetJWTService().SignJWT(nil)
	srv := &Server{log: stubLogger{}}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+tok))
	ctx = utils.SetLoggerToContext(ctx, stubLogger{})
	resp, err := srv.CreateItem(ctx, &pb.Item{Id: 1, Name: "x"})
	if err != nil || resp.GetName() != "x" {
		t.Fatalf("unexpected %v err %v", resp, err)
	}
}

func TestListGetUpdateDelete(t *testing.T) {
	defer monkey.UnpatchAll()
	patchRepo := monkey.Patch(repo.NewRepository, func(logger.Logger, db.DB) repo.Impl { return &repo.Repository{} })
	t.Cleanup(patchRepo.Unpatch)
	monkey.PatchInstanceMethod(reflect.TypeOf(&repo.Repository{}), "GetItemRepository", func(*repo.Repository) repoitem.Repository { return dummyItemRepo{} })
	monkey.Patch(outbound.NewOutbound, func(logger.Logger) outbound.Impl { return stubOutbound{} })

	dir := t.TempDir()
	priv, pub := writeKeyFiles(t, dir)
	config.Cfg = &config.Config{AppName: "APP", LogTarget: config.LogConfig{Path: dir, FileName: "app.log"}, Server: config.ServerConfig{JWT: config.JWTConfig{PrivateKey: priv, PublicKey: pub}}}

	if err := utils.InitializeJWTService(true, true); err != nil {
		t.Fatal(err)
	}
	tok, _ := utils.GetJWTService().SignJWT(nil)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+tok))
	ctx = utils.SetLoggerToContext(ctx, stubLogger{})
	srv := &Server{log: stubLogger{}}

	listResp, err := srv.ListItems(ctx, &pb.Empty{})
	if err != nil || len(listResp.GetItems()) != 1 {
		t.Fatalf("list error %v", err)
	}
	getResp, err := srv.GetItem(ctx, &pb.ItemID{Id: 1})
	if err != nil || getResp.GetName() != "b" {
		t.Fatalf("get error %v", err)
	}
	upResp, err := srv.UpdateItem(ctx, &pb.Item{Id: 2, Name: "b"})
	if err != nil || upResp.GetName() != "b" {
		t.Fatalf("update %#v err %v", upResp, err)
	}
	_, err = srv.DeleteItem(ctx, &pb.ItemID{Id: 3})
	if err != nil {
		t.Fatalf("delete err %v", err)
	}
}

func TestHandlersError(t *testing.T) {
	defer monkey.UnpatchAll()
	patchRepo := monkey.Patch(repo.NewRepository, func(logger.Logger, db.DB) repo.Impl { return &repo.Repository{} })
	t.Cleanup(patchRepo.Unpatch)
	monkey.PatchInstanceMethod(reflect.TypeOf(&repo.Repository{}), "GetItemRepository", func(*repo.Repository) repoitem.Repository { return dummyItemRepoErr{} })
	monkey.Patch(outbound.NewOutbound, func(logger.Logger) outbound.Impl { return stubOutbound{} })

	dir := t.TempDir()
	priv, pub := writeKeyFiles(t, dir)
	config.Cfg = &config.Config{AppName: "APP", LogTarget: config.LogConfig{Path: dir, FileName: "app.log"}, Server: config.ServerConfig{JWT: config.JWTConfig{PrivateKey: priv, PublicKey: pub}}}

	if err := utils.InitializeJWTService(true, true); err != nil {
		t.Fatal(err)
	}
	tok, _ := utils.GetJWTService().SignJWT(nil)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+tok))
	ctx = utils.SetLoggerToContext(ctx, stubLogger{})
	srv := &Server{log: stubLogger{}}

	if _, err := srv.CreateItem(ctx, &pb.Item{Id: 1, Name: "x"}); status.Code(err) != codes.Internal {
		t.Fatalf("unexpected create err %v", err)
	}
	if _, err := srv.ListItems(ctx, &pb.Empty{}); status.Code(err) != codes.Internal {
		t.Fatalf("unexpected list err %v", err)
	}
	if _, err := srv.GetItem(ctx, &pb.ItemID{Id: 1}); status.Code(err) != codes.NotFound {
		t.Fatalf("unexpected get err %v", err)
	}
	if _, err := srv.UpdateItem(ctx, &pb.Item{Id: 1}); status.Code(err) != codes.Internal {
		t.Fatalf("unexpected update err %v", err)
	}
	if _, err := srv.DeleteItem(ctx, &pb.ItemID{Id: 1}); status.Code(err) != codes.Internal {
		t.Fatalf("unexpected delete err %v", err)
	}
}

func TestHandlersUnauthenticated(t *testing.T) {
	defer monkey.UnpatchAll()
	patchRepo := monkey.Patch(repo.NewRepository, func(logger.Logger, db.DB) repo.Impl { return &repo.Repository{} })
	t.Cleanup(patchRepo.Unpatch)
	monkey.PatchInstanceMethod(reflect.TypeOf(&repo.Repository{}), "GetItemRepository", func(*repo.Repository) repoitem.Repository { return dummyItemRepo{} })
	monkey.Patch(outbound.NewOutbound, func(logger.Logger) outbound.Impl { return stubOutbound{} })

	dir := t.TempDir()
	priv, pub := writeKeyFiles(t, dir)
	config.Cfg = &config.Config{AppName: "APP", LogTarget: config.LogConfig{Path: dir, FileName: "app.log"}, Server: config.ServerConfig{JWT: config.JWTConfig{PrivateKey: priv, PublicKey: pub}}}

	if err := utils.InitializeJWTService(true, true); err != nil {
		t.Fatal(err)
	}
	srv := &Server{log: stubLogger{}}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "bad"))
	ctx = utils.SetLoggerToContext(ctx, stubLogger{})
	if _, err := srv.ListItems(ctx, &pb.Empty{}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected unauthenticated list got %v", err)
	}
	if _, err := srv.GetItem(ctx, &pb.ItemID{Id: 1}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected unauthenticated get got %v", err)
	}
	if _, err := srv.UpdateItem(ctx, &pb.Item{Id: 1}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected unauthenticated update got %v", err)
	}
	if _, err := srv.DeleteItem(ctx, &pb.ItemID{Id: 1}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected unauthenticated delete got %v", err)
	}
}

func TestCreateItemLoggerGenerated(t *testing.T) {
	defer monkey.UnpatchAll()
	patchRepo := monkey.Patch(repo.NewRepository, func(logger.Logger, db.DB) repo.Impl { return &repo.Repository{} })
	t.Cleanup(patchRepo.Unpatch)
	monkey.PatchInstanceMethod(reflect.TypeOf(&repo.Repository{}), "GetItemRepository", func(*repo.Repository) repoitem.Repository { return dummyItemRepo{} })
	monkey.Patch(outbound.NewOutbound, func(logger.Logger) outbound.Impl { return stubOutbound{} })

	dir := t.TempDir()
	priv, pub := writeKeyFiles(t, dir)
	config.Cfg = &config.Config{AppName: "APP", LogTarget: config.LogConfig{Path: dir, FileName: "app.log"}, Server: config.ServerConfig{JWT: config.JWTConfig{PrivateKey: priv, PublicKey: pub}}}

	if err := utils.InitializeJWTService(true, true); err != nil {
		t.Fatal(err)
	}
	tok, _ := utils.GetJWTService().SignJWT(nil)

	var gotP, gotC string
	var cnt int
	monkey.Patch(logger.NewLogger, func(cfg config.LogConfig, p, c string) (logger.Logger, error) {
		gotP = p
		gotC = c
		return stubLogger{}, nil
	})
	monkey.Patch(utils.UniqueIdByTime, func(uint64) (string, error) {
		cnt++
		if cnt == 1 {
			return "pid", nil
		}
		return "cid", nil
	})

	srv := &Server{log: stubLogger{}}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+tok))
	// do not set logger in context so server will create one
	resp, err := srv.CreateItem(ctx, &pb.Item{Id: 1, Name: "x"})
	if err != nil || resp.GetName() != "x" {
		t.Fatalf("unexpected %v err %v", resp, err)
	}
	if gotP != "-pid" || gotC != "cid" {
		t.Fatalf("logger not created with expected ids: %s %s", gotP, gotC)
	}
}
