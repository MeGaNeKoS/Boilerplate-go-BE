package config

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"project-template/infrastructure/enums"
)

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := dir + "/config.yaml"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadConfigSuccess(t *testing.T) {
	yaml := `AppName: APP
Version: "2"
ExternalService:
  Example:
    Host: http://example.com
Server:
  Host: 0.0.0.0
  Port: "80"
  Environment: dev
  Timeout:
    Server: 1
REST:
  Host: 0.0.0.0
  Port: "8080"
Kafka:
  Brokers: ["localhost:9092"]
  Topic: t
Database:
  Host: db
  Port: "3306"
  User: u
  Password: p
  DBName: test
  MigrationPath: file://migrations
  DirtyStrategy: retry
LogTarget:
  Path: /tmp
  FileName: app.log
`
	p := writeTempConfig(t, yaml)
	if err := LoadConfig(p); err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if Cfg.AppName != "APP" || Cfg.Version != "2" || Cfg.Server.Environment != "dev" || Cfg.Database.DirtyStrategy != enums.DirtyStrategyRetry {
		t.Fatalf("config not loaded: %#v", Cfg)
	}
}

func TestLoadConfigBadPath(t *testing.T) {
	if err := LoadConfig("/no/such/file"); err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	p := writeTempConfig(t, "invalid: : :")
	if err := LoadConfig(p); err == nil {
		t.Fatal("expected error")
	}
}

type errCloseFile struct {
	io.Reader
}

func (errCloseFile) Close() error { return errors.New("close error") }

func TestLoadConfigDecodeAndCloseError(t *testing.T) {
	orig := openFile
	openFile = func(string) (io.ReadCloser, error) {
		return errCloseFile{strings.NewReader("bad: :")}, nil
	}
	defer func() { openFile = orig }()
	err := LoadConfig("dummy")
	if err == nil || !strings.Contains(err.Error(), "error decoding config file") || !strings.Contains(err.Error(), "error closing file") {
		t.Fatalf("expected decode and close error, got %v", err)
	}
}

func TestLoadConfigCloseError(t *testing.T) {
	yaml := "AppName: a\nVersion: '1'"
	orig := openFile
	openFile = func(string) (io.ReadCloser, error) {
		return errCloseFile{strings.NewReader(yaml)}, nil
	}
	defer func() { openFile = orig }()
	if err := LoadConfig("dummy"); err == nil || !strings.Contains(err.Error(), "error closing config file") {
		t.Fatalf("expected close error, got %v", err)
	}
}
