package cmd

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strings"
	"testing"

	"project-template/pkg/logger"

	"github.com/doug-martin/goqu/v9"

	"project-template/infrastructure/config"
	"project-template/infrastructure/db"
	"project-template/infrastructure/utils"

	"github.com/bouk/monkey"
)

type stubDB struct{ err error }

func (s stubDB) New() error                        { return s.err }
func (stubDB) SetLogger(logger.Logger)             {}
func (stubDB) GetDB() *sql.DB                      { return nil }
func (stubDB) GetGoquDialect() goqu.DialectWrapper { return goqu.DialectWrapper{} }
func (stubDB) ExecContext(context.Context, string, ...interface{}) (sql.Result, error) {
	return nil, nil
}
func (stubDB) WithTransaction(_ context.Context, fn func(ctx context.Context) error) error {
	return fn(context.Background())
}
func (stubDB) BeginTx(_ context.Context, _ *sql.TxOptions) (*sql.Tx, error) {
	return nil, nil
}
func (stubDB) Close() error { return nil }
func (stubDB) Ping() error  { return nil }

func writeConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cfg := `AppName: APP
Server:
  Host: 0.0.0.0
  Port: "80"
  Timeout:
    Server: 1
Kafka:
  Brokers: ["b"]
REST:
  Host: 0.0.0.0
  Port: "8080"
Database:
  Host: db
  Port: "3306"
  User: u
  Password: p
  DBName: test
  MigrationPath: file://migrations
  DirtyStrategy: retry
LogTarget:
  Path: ` + dir + `
  FileName: app.log
`
	path := dir + "/config.yaml"
	if err := os.WriteFile(path, []byte(cfg), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestInitDependenciesSuccess(t *testing.T) {
	cfgPath := writeConfig(t)

	calledLogger := false
	patchLogger := monkey.Patch(logger.NewLogger, func(c config.LogConfig, p, cID string) (logger.Logger, error) {
		calledLogger = true
		return stubLog{}, nil
	})
	patchDB := monkey.Patch(db.GetDBInstance, func() db.DB { return stubDB{} })
	patchMig := monkey.Patch(db.RunMigrations, func(string) error { return nil })
	patchJWT := monkey.Patch(utils.InitializeJWTService, func(private, public bool) error { return nil })
	defer patchLogger.Unpatch()
	defer patchDB.Unpatch()
	defer patchMig.Unpatch()
	defer patchJWT.Unpatch()

	logr, err := initDependencies(cfgPath)
	if err != nil {
		t.Fatalf("init error: %v", err)
	}
	if _, ok := logr.(stubLog); !ok || !calledLogger {
		t.Fatalf("stub logger not used")
	}
}

func TestInitDependenciesErrors(t *testing.T) {
	// load config failure
	if _, err := initDependencies("/nope"); err == nil {
		t.Fatalf("expected load config error")
	}

	cfgPath := writeConfig(t)

	patchLogger := monkey.Patch(logger.NewLogger, func(c config.LogConfig, p, cID string) (logger.Logger, error) {
		return nil, errors.New("logger")
	})
	if _, err := initDependencies(cfgPath); err == nil || !strings.Contains(err.Error(), "logger") {
		patchLogger.Unpatch()
		t.Fatalf("unexpected err %v", err)
	}
	patchLogger.Unpatch()

	patchLogger = monkey.Patch(logger.NewLogger, func(c config.LogConfig, p, cID string) (logger.Logger, error) { return stubLog{}, nil })
	patchDB := monkey.Patch(db.GetDBInstance, func() db.DB { return stubDB{err: errors.New("db")} })
	if _, err := initDependencies(cfgPath); err == nil || !strings.Contains(err.Error(), "db connect") {
		patchLogger.Unpatch()
		patchDB.Unpatch()
		t.Fatalf("unexpected err %v", err)
	}
	patchLogger.Unpatch()
	patchDB.Unpatch()

	patchLogger = monkey.Patch(logger.NewLogger, func(c config.LogConfig, p, cID string) (logger.Logger, error) { return stubLog{}, nil })
	patchDB = monkey.Patch(db.GetDBInstance, func() db.DB { return stubDB{} })
	patchMig := monkey.Patch(db.RunMigrations, func(string) error { return nil })
	patchJWT := monkey.Patch(utils.InitializeJWTService, func(private, public bool) error { return errors.New("jwt") })
	if _, err := initDependencies(cfgPath); err == nil || !strings.Contains(err.Error(), "init jwt") {
		patchLogger.Unpatch()
		patchDB.Unpatch()
		patchMig.Unpatch()
		patchJWT.Unpatch()
		t.Fatalf("unexpected err %v", err)
	}

	patchLogger.Unpatch()
	patchDB.Unpatch()
	patchMig.Unpatch()
	patchJWT.Unpatch()
}

func TestInitDependenciesMigrateError(t *testing.T) {
	cfgPath := writeConfig(t)

	patchLogger := monkey.Patch(logger.NewLogger, func(c config.LogConfig, p, cID string) (logger.Logger, error) { return stubLog{}, nil })
	patchDB := monkey.Patch(db.GetDBInstance, func() db.DB { return stubDB{} })
	patchMig := monkey.Patch(db.RunMigrations, func(string) error { return errors.New("boom") })
	patchJWT := monkey.Patch(utils.InitializeJWTService, func(private, public bool) error { return nil })

	if _, err := initDependencies(cfgPath); err == nil || !strings.Contains(err.Error(), "db migrate") {
		patchLogger.Unpatch()
		patchDB.Unpatch()
		patchMig.Unpatch()
		patchJWT.Unpatch()
		t.Fatalf("unexpected err %v", err)
	}

	patchLogger.Unpatch()
	patchDB.Unpatch()
	patchMig.Unpatch()
	patchJWT.Unpatch()
}
