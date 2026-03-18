package middleware

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bouk/monkey"
	"github.com/golang-jwt/jwt/v4"

	"project-template/infrastructure/config"
	"project-template/infrastructure/db"
	"project-template/infrastructure/dto/response"
	"project-template/infrastructure/utils"
	"project-template/outbound"
	"project-template/outbound/service/example"
	"project-template/pkg/logger"
	repo "project-template/repositories"
)

type apiStubLogger struct{}

func (apiStubLogger) Debug(string)                  {}
func (apiStubLogger) DebugF(string, ...interface{}) {}
func (apiStubLogger) Info(string)                   {}
func (apiStubLogger) InfoF(string, ...interface{})  {}
func (apiStubLogger) Warn(string)                   {}
func (apiStubLogger) WarnF(string, ...interface{})  {}
func (apiStubLogger) Error(string)                  {}
func (apiStubLogger) ErrorF(string, ...interface{}) {}
func (apiStubLogger) Fatal(string)                  {}
func (apiStubLogger) FatalF(string, ...interface{}) {}
func (apiStubLogger) ParentID() string              { return "p" }
func (apiStubLogger) ChildID() string               { return "c" }
func (apiStubLogger) CloseLogFile()                 {}

type stubExampleAgg struct{}

func (stubExampleAgg) HTTP() example.Outbound       { return nil }
func (stubExampleAgg) GRPC() example.GrpcOutbound   { return nil }
func (stubExampleAgg) Kafka() example.KafkaOutbound { return nil }

type stubOutbound struct{}

func (stubOutbound) Example() example.Service { return stubExampleAgg{} }

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

// testJWTService implements utils.JWTService for testing with real RSA keys.
type testJWTService struct {
	publicKey *rsa.PublicKey
}

func (s *testJWTService) SignJWT(map[string]interface{}) (string, error) { return "", nil }
func (s *testJWTService) GenerateRefreshToken(int) (string, error)       { return "", nil }
func (s *testJWTService) ParseJWT(tokenString string, claims jwt.Claims) error {
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return s.publicKey, nil
	})
	if err != nil {
		return err
	}
	if !token.Valid {
		return &jwt.ValidationError{Errors: jwt.ValidationErrorClaimsInvalid}
	}
	return nil
}

func createTestJWTService(t *testing.T, privPath, pubPath string) utils.JWTService {
	t.Helper()
	pubData, err := os.ReadFile(pubPath)
	if err != nil {
		t.Fatal(err)
	}
	pubKey, err := jwt.ParseRSAPublicKeyFromPEM(pubData)
	if err != nil {
		t.Fatal(err)
	}
	return &testJWTService{publicKey: pubKey}
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
		if repoCtx := utils.GetRepoCtx(r.Context()); repoCtx == nil {
			t.Fatal("repoCtx missing")
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
	config.Cfg = &config.Config{AppName: "APP"}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next should not be called")
	})
	mw := AuthMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req = req.WithContext(utils.SetLoggerToContext(req.Context(), apiStubLogger{}))
	req.Header.Set("Authorization", "Bearer badtoken")
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 got %d", rec.Code)
	}
	wwwAuth := rec.Header().Get("WWW-Authenticate")
	if wwwAuth == "" {
		t.Fatal("missing WWW-Authenticate header")
	}
	if !strings.Contains(wwwAuth, `realm="APP"`) {
		t.Fatalf("WWW-Authenticate missing realm: %s", wwwAuth)
	}
	if !strings.Contains(wwwAuth, `error="invalid_token"`) {
		t.Fatalf("WWW-Authenticate missing error: %s", wwwAuth)
	}
}

func TestAuthMiddlewareMissingToken(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next should not be called")
	})
	mw := AuthMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req = req.WithContext(utils.SetLoggerToContext(req.Context(), apiStubLogger{}))
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 got %d", rec.Code)
	}
	wwwAuth := rec.Header().Get("WWW-Authenticate")
	if wwwAuth == "" {
		t.Fatal("missing WWW-Authenticate header")
	}
	if !strings.Contains(wwwAuth, `realm="APP"`) {
		t.Fatalf("WWW-Authenticate missing realm: %s", wwwAuth)
	}
	// Missing token should not include error parameter
	if strings.Contains(wwwAuth, `error=`) {
		t.Fatalf("WWW-Authenticate should not have error for missing token: %s", wwwAuth)
	}
}

func TestAuthMiddlewareEmptyBearer(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next should not be called")
	})
	mw := AuthMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req = req.WithContext(utils.SetLoggerToContext(req.Context(), apiStubLogger{}))
	req.Header.Set("Authorization", "Bearer ")
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 got %d", rec.Code)
	}
	wwwAuth := rec.Header().Get("WWW-Authenticate")
	if wwwAuth == "" {
		t.Fatal("missing WWW-Authenticate header")
	}
}

func TestAuthMiddlewareExpiredToken(t *testing.T) {
	defer monkey.UnpatchAll()

	dir := t.TempDir()
	priv, pub := writeKeyFiles(t, dir)
	config.Cfg = &config.Config{AppName: "APP", Server: config.ServerConfig{Environment: "dev", JWT: config.JWTConfig{PrivateKey: priv, PublicKey: pub}}, LogTarget: config.LogConfig{Path: dir, FileName: "app.log"}}

	privKeyData, _ := os.ReadFile(priv)
	privateKey, _ := jwt.ParseRSAPrivateKeyFromPEM(privKeyData)

	// Create an expired token
	claims := jwt.MapClaims{
		"exp":         time.Now().Add(-time.Hour).Unix(),
		"iss":         "APP",
		"environment": "dev",
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenStr, err := tok.SignedString(privateKey)
	if err != nil {
		t.Fatal(err)
	}

	// Patch GetJWTService to use our keys
	svc := createTestJWTService(t, priv, pub)
	monkey.Patch(utils.GetJWTService, func() utils.JWTService { return svc })

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next should not be called")
	})
	mw := AuthMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req = req.WithContext(utils.SetLoggerToContext(req.Context(), apiStubLogger{}))
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 got %d", rec.Code)
	}
	wwwAuth := rec.Header().Get("WWW-Authenticate")
	if !strings.Contains(wwwAuth, `error="invalid_token"`) {
		t.Fatalf("WWW-Authenticate missing error: %s", wwwAuth)
	}
	if !strings.Contains(wwwAuth, "expired") {
		t.Fatalf("WWW-Authenticate missing expired description: %s", wwwAuth)
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

func TestRespondNonJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	resp := &response.HttpResponse{HTTPCode: http.StatusOK, ContentType: "text/plain", RawResponsePayload: []byte("hi")}
	respond(rec, resp)
	if rec.Code != http.StatusOK {
		t.Fatalf("code got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/plain" {
		t.Fatalf("content type %q", ct)
	}
	if rec.Body.String() != "hi" {
		t.Fatalf("body %q", rec.Body.String())
	}
}
