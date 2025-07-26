package utils

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"project-template/infrastructure/config"

	"github.com/golang-jwt/jwt/v4"
)

type failingReader struct{}

func (failingReader) Read(_ []byte) (int, error) { return 0, io.ErrUnexpectedEOF }

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

func TestGenerateRefreshToken(t *testing.T) {
	svc := &jwtService{}
	old := randomReader
	defer func() { randomReader = old }()
	randomReader = bytes.NewBuffer([]byte("abcdefghijklmnop"))
	tok, err := svc.GenerateRefreshToken(8)
	if err != nil {
		t.Fatalf("GenerateRefreshToken error: %v", err)
	}
	if tok != "YWJjZGVm" { // base64 of "abcdefgh" starts with this
		t.Fatalf("unexpected token %s", tok)
	}
}

func TestGenerateRefreshTokenError(t *testing.T) {
	svc := &jwtService{}
	old := randomReader
	defer func() { randomReader = old }()
	randomReader = failingReader{}
	if _, err := svc.GenerateRefreshToken(5); err == nil {
		t.Fatal("expected error")
	}
}

func TestClaimsMarshalJSON(t *testing.T) {
	c := Claims{Environment: "dev", RegisteredClaims: jwt.RegisteredClaims{Issuer: "app", ExpiresAt: jwt.NewNumericDate(jwt.TimeFunc())}, Additional: map[string]interface{}{"k": "v"}}
	b, err := c.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if m["environment"] != "dev" || m["k"] != "v" {
		t.Fatalf("unexpected map %#v", m)
	}
}

func TestJWTSignAndParse(t *testing.T) {
	dir := t.TempDir()
	priv, pub := writeKeyFiles(t, dir)

	config.Cfg = &config.Config{AppName: "APP", Server: config.ServerConfig{Environment: "dev", JWT: config.JWTConfig{PrivateKey: priv, PublicKey: pub}}}

	jwtInstance = nil
	jwtOnce = sync.Once{}
	if err := InitializeJWTService(true, true); err != nil {
		t.Fatalf("init err: %v", err)
	}
	svc := GetJWTService()

	token, err := svc.SignJWT(map[string]interface{}{"foo": "bar"})
	if err != nil {
		t.Fatalf("sign err: %v", err)
	}

	var claims Claims
	if err := svc.ParseJWT(token, &claims); err != nil {
		t.Fatalf("parse err: %v", err)
	}
	if claims.Environment != "dev" {
		t.Fatalf("unexpected environment %s", claims.Environment)
	}
}

func TestParseJWTEnvMismatch(t *testing.T) {
	dir := t.TempDir()
	priv, pub := writeKeyFiles(t, dir)
	config.Cfg = &config.Config{AppName: "APP", Server: config.ServerConfig{Environment: "dev", JWT: config.JWTConfig{PrivateKey: priv, PublicKey: pub}}}
	jwtInstance = nil
	jwtOnce = sync.Once{}
	if err := InitializeJWTService(true, true); err != nil {
		t.Fatalf("init err: %v", err)
	}
	svc := GetJWTService()
	token, err := svc.SignJWT(nil)
	if err != nil {
		t.Fatalf("sign err: %v", err)
	}
	config.Cfg.Server.Environment = "prod"
	if err := svc.ParseJWT(token, &Claims{}); err == nil {
		t.Fatalf("expected environment error")
	}
}

func TestNewJWTServiceErrors(t *testing.T) {
	dir := t.TempDir()
	config.Cfg = &config.Config{Server: config.ServerConfig{JWT: config.JWTConfig{PrivateKey: filepath.Join(dir, "no"), PublicKey: filepath.Join(dir, "no")}}}
	if _, err := newJWTService(true, true); err == nil {
		t.Fatalf("expected error")
	}

	config.Cfg.Server.JWT.PublicKey = filepath.Join(dir, "missing")
	if _, err := newJWTService(false, true); err == nil || !strings.Contains(err.Error(), "read public key") {
		t.Fatalf("expected read public key error")
	}

	badPriv := filepath.Join(dir, "badpriv.pem")
	os.WriteFile(badPriv, []byte("BAD"), 0600)
	config.Cfg.Server.JWT.PrivateKey = badPriv
	if _, err := newJWTService(true, false); err == nil || !strings.Contains(err.Error(), "parse private key") {
		t.Fatalf("expected parse private key error")
	}

	badPub := filepath.Join(dir, "badpub.pem")
	os.WriteFile(badPub, []byte("BAD"), 0600)
	config.Cfg.Server.JWT.PublicKey = badPub
	if _, err := newJWTService(false, true); err == nil || !strings.Contains(err.Error(), "parse public key") {
		t.Fatalf("expected parse public key error")
	}
}

func TestGetJWTServicePanic(t *testing.T) {
	jwtInstance = nil
	jwtOnce = sync.Once{}
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	GetJWTService()
}

func TestSignParseErrors(t *testing.T) {
	svc := &jwtService{}
	if _, err := svc.SignJWT(nil); err == nil {
		t.Fatalf("expected error signing without key")
	}
	if err := svc.ParseJWT("bad", &Claims{}); err == nil {
		t.Fatalf("expected parse error")
	}
}

func TestSignJWTSigningFailure(t *testing.T) {
	svc := &jwtService{privateKey: &rsa.PrivateKey{}}
	if _, err := svc.SignJWT(nil); err == nil {
		t.Fatalf("expected signing failure")
	}
}

func TestParseJWTUnexpectedMethod(t *testing.T) {
	dir := t.TempDir()
	priv, pub := writeKeyFiles(t, dir)
	config.Cfg = &config.Config{AppName: "APP", Server: config.ServerConfig{Environment: "dev", JWT: config.JWTConfig{PrivateKey: priv, PublicKey: pub}}}
	jwtInstance = nil
	jwtOnce = sync.Once{}
	if err := InitializeJWTService(true, true); err != nil {
		t.Fatalf("init err: %v", err)
	}
	svc := GetJWTService()
	claims := jwt.MapClaims{"environment": "dev"}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.ParseJWT(signed, &Claims{}); err == nil || !strings.Contains(err.Error(), "unexpected signing method") {
		t.Fatalf("expected unexpected signing method error, got %v", err)
	}
}

func TestParseJWTEnvWrongKind(t *testing.T) {
	dir := t.TempDir()
	priv, pub := writeKeyFiles(t, dir)
	config.Cfg = &config.Config{AppName: "APP", Server: config.ServerConfig{Environment: "dev", JWT: config.JWTConfig{PrivateKey: priv, PublicKey: pub}}}
	jwtInstance = nil
	jwtOnce = sync.Once{}
	if err := InitializeJWTService(true, true); err != nil {
		t.Fatalf("init err: %v", err)
	}
	svc := GetJWTService()
	token, _ := svc.SignJWT(nil)
	type ifaceEnv struct {
		Environment interface{}
		jwt.RegisteredClaims
	}
	if err := svc.ParseJWT(token, &ifaceEnv{}); err == nil || !strings.Contains(err.Error(), "environment field is not a string") {
		t.Fatalf("expected env kind error, got %v", err)
	}
}

func TestParseJWTInvalidToken(t *testing.T) {
	svc := &jwtService{publicKey: &rsa.PublicKey{}}
	old := parseWithClaims
	defer func() { parseWithClaims = old }()
	parseWithClaims = func(tokenString string, claims jwt.Claims, keyFunc jwt.Keyfunc, options ...jwt.ParserOption) (*jwt.Token, error) {
		return &jwt.Token{Valid: false, Method: jwt.SigningMethodRS256}, nil
	}
	if err := svc.ParseJWT("x", &Claims{}); err == nil || !strings.Contains(err.Error(), "invalid token") {
		t.Fatalf("expected invalid token error")
	}
}

func TestParseJWTMissingField(t *testing.T) {
	dir := t.TempDir()
	priv, pub := writeKeyFiles(t, dir)
	config.Cfg = &config.Config{AppName: "APP", Server: config.ServerConfig{Environment: "dev", JWT: config.JWTConfig{PrivateKey: priv, PublicKey: pub}}}
	jwtInstance = nil
	jwtOnce = sync.Once{}
	if err := InitializeJWTService(true, true); err != nil {
		t.Fatalf("init err: %v", err)
	}
	svc := GetJWTService()
	token, _ := svc.SignJWT(nil)
	type noEnv struct{ jwt.RegisteredClaims }
	if err := svc.ParseJWT(token, &noEnv{}); err == nil || !strings.Contains(err.Error(), "environment field is missing") {
		t.Fatalf("expected env missing error, got %v", err)
	}
}

func TestParseJWTWrongType(t *testing.T) {
	dir := t.TempDir()
	priv, pub := writeKeyFiles(t, dir)
	config.Cfg = &config.Config{AppName: "APP", Server: config.ServerConfig{Environment: "dev", JWT: config.JWTConfig{PrivateKey: priv, PublicKey: pub}}}
	jwtInstance = nil
	jwtOnce = sync.Once{}
	if err := InitializeJWTService(true, true); err != nil {
		t.Fatalf("init err: %v", err)
	}
	svc := GetJWTService()
	token, _ := svc.SignJWT(nil)
	type badEnv struct {
		Environment int
		jwt.RegisteredClaims
	}
	if err := svc.ParseJWT(token, &badEnv{}); err == nil || !strings.Contains(err.Error(), "cannot unmarshal") {
		t.Fatalf("expected env type error, got %v", err)
	}
}
