package middleware

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/bouk/monkey"

	"project-template/infrastructure/config"
	"project-template/infrastructure/db"
	models "project-template/infrastructure/dto/item"
	"project-template/infrastructure/utils"
	"project-template/outbound"
	"project-template/outbound/service/example"
	"project-template/pkg/logger"
	repo "project-template/repositories"
	repoitem "project-template/repositories/item"
)

type apiStubLogger struct{}

func (apiStubLogger) DebugF(string, ...interface{}) {}
func (apiStubLogger) InfoF(string, ...interface{})  {}
func (apiStubLogger) WarnF(string, ...interface{})  {}
func (apiStubLogger) ErrorF(string, ...interface{}) {}
func (apiStubLogger) FatalF(string, ...interface{}) {}
func (apiStubLogger) ParentID() string              { return "p" }
func (apiStubLogger) ChildID() string               { return "c" }
func (apiStubLogger) CloseLogFile()                 {}

type stubRepo struct{}

func (stubRepo) GetItemRepository() repoitem.Repository { return nil }

type stubExampleAgg struct{}

func (stubExampleAgg) HTTP() example.ExampleOutbound       { return nil }
func (stubExampleAgg) GRPC() example.ExampleGRPCOutbound   { return nil }
func (stubExampleAgg) Kafka() example.ExampleKafkaOutbound { return nil }

type stubOutbound struct{}

func (stubOutbound) Example() example.Service { return stubExampleAgg{} }

type stubService struct{}

func (stubService) CreateItem(context.Context, models.Item) (models.Item, error) {
	return models.Item{}, nil
}
func (stubService) ListItems(context.Context) ([]models.Item, error) { return nil, nil }
func (stubService) GetItem(context.Context, int) (models.Item, error) {
	return models.Item{}, nil
}
func (stubService) UpdateItem(context.Context, models.Item) (models.Item, error) {
	return models.Item{}, nil
}
func (stubService) DeleteItem(context.Context, int) error { return nil }

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

// Test ServiceMiddleware building handler and AuthMiddleware parsing token.
func TestMiddlewares(t *testing.T) {
	defer monkey.UnpatchAll()
	patchRepo := monkey.Patch(repo.NewRepository, func(logger.Logger, db.DB) repo.Impl { return &repo.Repository{} })
	t.Cleanup(patchRepo.Unpatch)
	monkey.Patch(outbound.NewOutbound, func(logger.Logger) outbound.Impl { return stubOutbound{} })

	dir := t.TempDir()
	priv, pub := writeKeyFiles(t, dir)
	config.Cfg = &config.Config{AppName: "APP", Server: config.ServerConfig{Environment: "dev", JWT: config.JWTConfig{PrivateKey: priv, PublicKey: pub}}, LogTarget: config.LogConfig{Path: dir, FileName: "app.log"}}
	if err := utils.InitializeJWTService(true, true); err != nil {
		t.Fatal(err)
	}
	tok, _ := utils.GetJWTService().SignJWT(nil)

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if repo := utils.GetRepoCtx(r.Context()); repo == nil {
			t.Fatal("repo missing")
		}
		if out := utils.GetOutboundCtx(r.Context()); out == nil {
			t.Fatal("outbound missing")
		}
		if u, ok := utils.GetUserCtx(r.Context()); !ok || u == nil {
			t.Fatal("user missing")
		}
	})

	mw := ServiceMiddleware(AuthMiddleware(next))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(utils.SetLoggerToContext(req.Context(), apiStubLogger{}))
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)

	if !called {
		t.Fatal("next not called")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected code %d", rec.Code)
	}
}

func TestAuthMiddlewareInvalidToken(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next should not be called")
	})
	mw := AuthMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(utils.SetLoggerToContext(req.Context(), apiStubLogger{}))
	req.Header.Set("Authorization", "bad")
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 got %d", rec.Code)
	}
}

func TestAuthMiddlewareNoLogger(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next should not be called")
	})
	mw := AuthMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("want 500 got %d", rec.Code)
	}
}
