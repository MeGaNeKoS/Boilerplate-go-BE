package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/bouk/monkey"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	mysqlmigrate "github.com/golang-migrate/migrate/v4/database/mysql"
	"project-template/infrastructure/config"
	"project-template/infrastructure/enums"
)

type stubLogger struct {
	fatalMsgs []string
	infoMsgs  []string
}

func (s *stubLogger) DebugF(string, ...interface{}) {}
func (s *stubLogger) InfoF(format string, args ...interface{}) {
	s.infoMsgs = append(s.infoMsgs, fmt.Sprintf(format, args...))
}
func (s *stubLogger) WarnF(string, ...interface{})  {}
func (s *stubLogger) ErrorF(string, ...interface{}) {}
func (s *stubLogger) FatalF(format string, args ...interface{}) {
	s.fatalMsgs = append(s.fatalMsgs, fmt.Sprintf(format, args...))
}
func (s *stubLogger) ParentID() string { return "" }
func (s *stubLogger) ChildID() string  { return "" }

//---- tests for db.init.go extra branches ----

func TestSetLoggerAndGetDBWithLogger(t *testing.T) {
	impl := &dbImpl{}
	l := &stubLogger{}
	impl.SetLogger(l)
	if impl.logger != l {
		t.Fatalf("logger not set")
	}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unexpected panic: %v", r)
		}
	}()
	_ = impl.GetDB()
	if len(l.fatalMsgs) == 0 || !strings.Contains(l.fatalMsgs[0], "Database connection is not initialized") {
		t.Fatalf("fatal not called")
	}
}

func TestNewOpenErrorWithLogger(t *testing.T) {
	impl := &dbImpl{}
	l := &stubLogger{}
	impl.SetLogger(l)
	stubConfig()
	db, _, _ := sqlmock.New()
	monkey.Patch(sql.Open, func(string, string) (*sql.DB, error) { return db, errors.New("open") })
	defer monkey.Unpatch(sql.Open)
	if err := impl.New(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(l.fatalMsgs) != 1 || !strings.Contains(l.fatalMsgs[0], "Error connecting") {
		t.Fatalf("fatal for open not called")
	}
}

func TestNewPingErrorsWithLogger(t *testing.T) {
	impl := &dbImpl{}
	l := &stubLogger{}
	impl.SetLogger(l)
	stubConfig()
	db, _, _ := sqlmock.New()
	monkey.Patch(sql.Open, func(string, string) (*sql.DB, error) { return db, nil })
	defer monkey.Unpatch(sql.Open)
	monkey.PatchInstanceMethod(reflect.TypeOf(&sql.DB{}), "PingContext", func(*sql.DB, context.Context) error { return errors.New("ping") })
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(&sql.DB{}), "PingContext")
	monkey.PatchInstanceMethod(reflect.TypeOf(&sql.DB{}), "Close", func(*sql.DB) error { return errors.New("close") })
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(&sql.DB{}), "Close")
	if err := impl.New(); err == nil {
		t.Fatalf("expected error")
	}
	if len(l.fatalMsgs) != 2 {
		t.Fatalf("expected two fatal calls")
	}
	if !strings.Contains(l.fatalMsgs[0], "Error closing") || !strings.Contains(l.fatalMsgs[1], "Lost connection") {
		t.Fatalf("unexpected fatal messages %v", l.fatalMsgs)
	}
	if impl.db != nil {
		t.Fatalf("db should be nil")
	}
}

func TestNewSuccessWithLogger(t *testing.T) {
	impl := &dbImpl{}
	l := &stubLogger{}
	impl.SetLogger(l)
	stubConfig()
	db, _, _ := sqlmock.New()
	monkey.Patch(sql.Open, func(string, string) (*sql.DB, error) { return db, nil })
	defer monkey.Unpatch(sql.Open)
	monkey.PatchInstanceMethod(reflect.TypeOf(&sql.DB{}), "PingContext", func(*sql.DB, context.Context) error { return nil })
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(&sql.DB{}), "PingContext")
	if err := impl.New(); err != nil {
		t.Fatalf("unexpected %v", err)
	}
	if len(l.infoMsgs) != 1 {
		t.Fatalf("info not called")
	}
	if impl.db != db {
		t.Fatalf("db not set")
	}
}

//---- additional RunMigrations branches ----

func TestRunMigrationsDefaultStrategy(t *testing.T) {
	sqlDB, _, _ := sqlmock.New()
	patchDB := monkey.Patch(GetDBInstance, func() DB { return testDB{sqlDB} })
	defer patchDB.Unpatch()
	patchDriver := monkey.Patch(mysqlmigrate.WithInstance, func(*sql.DB, *mysqlmigrate.Config) (database.Driver, error) { return nil, nil })
	defer patchDriver.Unpatch()
	patchNew := monkey.Patch(migrate.NewWithDatabaseInstance, func(string, string, database.Driver) (*migrate.Migrate, error) { return &migrate.Migrate{}, nil })
	defer patchNew.Unpatch()
	patchUp := monkey.PatchInstanceMethod(reflect.TypeOf(&migrate.Migrate{}), "Up", func(*migrate.Migrate) error { return nil })
	defer patchUp.Unpatch()
	saved := config.Cfg
	config.Cfg = &config.Config{Database: config.DatabaseConfig{DirtyStrategy: ""}}
	defer func() { config.Cfg = saved }()
	if err := RunMigrations("file://migrations"); err != nil {
		t.Fatalf("unexpected %v", err)
	}
}

func TestRunMigrationsForceError(t *testing.T) {
	sqlDB, _, _ := sqlmock.New()
	patchDB := monkey.Patch(GetDBInstance, func() DB { return testDB{sqlDB} })
	defer patchDB.Unpatch()
	patchDriver := monkey.Patch(mysqlmigrate.WithInstance, func(*sql.DB, *mysqlmigrate.Config) (database.Driver, error) { return nil, nil })
	defer patchDriver.Unpatch()
	patchNew := monkey.Patch(migrate.NewWithDatabaseInstance, func(string, string, database.Driver) (*migrate.Migrate, error) { return &migrate.Migrate{}, nil })
	defer patchNew.Unpatch()
	patchUp := monkey.PatchInstanceMethod(reflect.TypeOf(&migrate.Migrate{}), "Up", func(*migrate.Migrate) error { return migrate.ErrDirty{Version: 1} })
	defer patchUp.Unpatch()
	patchForce := monkey.PatchInstanceMethod(reflect.TypeOf(&migrate.Migrate{}), "Force", func(*migrate.Migrate, int) error { return errors.New("force") })
	defer patchForce.Unpatch()
	saved := config.Cfg
	config.Cfg = &config.Config{Database: config.DatabaseConfig{DirtyStrategy: enums.DirtyStrategyRetry}}
	defer func() { config.Cfg = saved }()
	if err := RunMigrations("file://migrations"); err == nil || !strings.Contains(err.Error(), "force dirty") {
		t.Fatalf("expected force error, got %v", err)
	}
}

func TestRunMigrationsStepError(t *testing.T) {
	sqlDB, _, _ := sqlmock.New()
	patchDB := monkey.Patch(GetDBInstance, func() DB { return testDB{sqlDB} })
	defer patchDB.Unpatch()
	patchDriver := monkey.Patch(mysqlmigrate.WithInstance, func(*sql.DB, *mysqlmigrate.Config) (database.Driver, error) { return nil, nil })
	defer patchDriver.Unpatch()
	patchNew := monkey.Patch(migrate.NewWithDatabaseInstance, func(string, string, database.Driver) (*migrate.Migrate, error) { return &migrate.Migrate{}, nil })
	defer patchNew.Unpatch()
	cnt := 0
	patchUp := monkey.PatchInstanceMethod(reflect.TypeOf(&migrate.Migrate{}), "Up", func(*migrate.Migrate) error {
		if cnt == 0 {
			cnt++
			return migrate.ErrDirty{Version: 1}
		}
		return nil
	})
	defer patchUp.Unpatch()
	patchForce := monkey.PatchInstanceMethod(reflect.TypeOf(&migrate.Migrate{}), "Force", func(*migrate.Migrate, int) error { return nil })
	defer patchForce.Unpatch()
	patchSteps := monkey.PatchInstanceMethod(reflect.TypeOf(&migrate.Migrate{}), "Steps", func(*migrate.Migrate, int) error { return errors.New("step") })
	defer patchSteps.Unpatch()
	saved := config.Cfg
	config.Cfg = &config.Config{Database: config.DatabaseConfig{DirtyStrategy: enums.DirtyStrategyRetry}}
	defer func() { config.Cfg = saved }()
	if err := RunMigrations("file://migrations"); err == nil || !strings.Contains(err.Error(), "rollback dirty") {
		t.Fatalf("expected step error, got %v", err)
	}
}

func TestRunMigrationsUpAfterForceError(t *testing.T) {
	sqlDB, _, _ := sqlmock.New()
	patchDB := monkey.Patch(GetDBInstance, func() DB { return testDB{sqlDB} })
	defer patchDB.Unpatch()
	patchDriver := monkey.Patch(mysqlmigrate.WithInstance, func(*sql.DB, *mysqlmigrate.Config) (database.Driver, error) { return nil, nil })
	defer patchDriver.Unpatch()
	patchNew := monkey.Patch(migrate.NewWithDatabaseInstance, func(string, string, database.Driver) (*migrate.Migrate, error) { return &migrate.Migrate{}, nil })
	defer patchNew.Unpatch()
	cnt := 0
	patchUp := monkey.PatchInstanceMethod(reflect.TypeOf(&migrate.Migrate{}), "Up", func(*migrate.Migrate) error {
		if cnt == 0 {
			cnt++
			return migrate.ErrDirty{Version: 1}
		}
		return errors.New("after")
	})
	defer patchUp.Unpatch()
	patchForce := monkey.PatchInstanceMethod(reflect.TypeOf(&migrate.Migrate{}), "Force", func(*migrate.Migrate, int) error { return nil })
	defer patchForce.Unpatch()
	patchSteps := monkey.PatchInstanceMethod(reflect.TypeOf(&migrate.Migrate{}), "Steps", func(*migrate.Migrate, int) error { return nil })
	defer patchSteps.Unpatch()
	saved := config.Cfg
	config.Cfg = &config.Config{Database: config.DatabaseConfig{DirtyStrategy: enums.DirtyStrategyRetry}}
	defer func() { config.Cfg = saved }()
	if err := RunMigrations("file://migrations"); err == nil || !strings.Contains(err.Error(), "migrate up after force") {
		t.Fatalf("expected error, got %v", err)
	}
}
