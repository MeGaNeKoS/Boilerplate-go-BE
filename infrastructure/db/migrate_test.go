package db

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"

	"project-template/infrastructure/config"
	"project-template/infrastructure/enums"
	"project-template/pkg/logger"
	"strings"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/bouk/monkey"
	"github.com/doug-martin/goqu/v9"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	mysqlmigrate "github.com/golang-migrate/migrate/v4/database/mysql"
)

func TestRunMigrationsSuccess(t *testing.T) {
	sqlDB, _, _ := sqlmock.New()
	patchDB := monkey.Patch(GetDBInstance, func() DB { return testDB{sqlDB} })
	patchDriver := monkey.Patch(mysqlmigrate.WithInstance, func(*sql.DB, *mysqlmigrate.Config) (database.Driver, error) {
		return nil, nil
	})
	patchNew := monkey.Patch(migrate.NewWithDatabaseInstance, func(string, string, database.Driver) (*migrate.Migrate, error) {
		return &migrate.Migrate{}, nil
	})
	patchUp := monkey.PatchInstanceMethod(reflect.TypeOf(&migrate.Migrate{}), "Up", func(*migrate.Migrate) error { return nil })
	saved := config.Cfg
	config.Cfg = &config.Config{Database: config.DatabaseConfig{DirtyStrategy: enums.DirtyStrategyRetry}}
	defer func() { config.Cfg = saved }()
	defer patchDB.Unpatch()
	defer patchDriver.Unpatch()
	defer patchNew.Unpatch()
	defer patchUp.Unpatch()

	if err := RunMigrations("file://migrations"); err != nil {
		t.Fatalf("unexpected %v", err)
	}
}

func TestRunMigrationsErrors(t *testing.T) {
	sqlDB, _, _ := sqlmock.New()
	patchDB := monkey.Patch(GetDBInstance, func() DB { return testDB{sqlDB} })
	defer patchDB.Unpatch()
	saved := config.Cfg
	config.Cfg = &config.Config{Database: config.DatabaseConfig{DirtyStrategy: enums.DirtyStrategyRetry}}
	defer func() { config.Cfg = saved }()

	// driver error
	patchDriver := monkey.Patch(mysqlmigrate.WithInstance, func(*sql.DB, *mysqlmigrate.Config) (database.Driver, error) {
		return nil, errors.New("drv")
	})
	if err := RunMigrations("file://migrations"); err == nil || !strings.Contains(err.Error(), "driver") {
		t.Fatalf("expected driver error, got %v", err)
	}
	patchDriver.Unpatch()

	// init error
	patchDriver = monkey.Patch(mysqlmigrate.WithInstance, func(*sql.DB, *mysqlmigrate.Config) (database.Driver, error) { return nil, nil })
	patchNew := monkey.Patch(migrate.NewWithDatabaseInstance, func(string, string, database.Driver) (*migrate.Migrate, error) {
		return nil, errors.New("init")
	})
	if err := RunMigrations("file://migrations"); err == nil || !strings.Contains(err.Error(), "migrate init") {
		t.Fatalf("expected init err, got %v", err)
	}
	patchNew.Unpatch()
	patchDriver.Unpatch()

	// up error not ErrNoChange
	patchDriver = monkey.Patch(mysqlmigrate.WithInstance, func(*sql.DB, *mysqlmigrate.Config) (database.Driver, error) { return nil, nil })
	patchNew = monkey.Patch(migrate.NewWithDatabaseInstance, func(string, string, database.Driver) (*migrate.Migrate, error) {
		return &migrate.Migrate{}, nil
	})
	patchUp := monkey.PatchInstanceMethod(reflect.TypeOf(&migrate.Migrate{}), "Up", func(*migrate.Migrate) error { return errors.New("up") })
	if err := RunMigrations("file://migrations"); err == nil || !strings.Contains(err.Error(), "migrate up") {
		t.Fatalf("expected up err, got %v", err)
	}
	patchUp.Unpatch()
	patchNew.Unpatch()

	// dirty db auto-fix
	cnt := 0
	var forced int
	var steps int
	patchNew = monkey.Patch(migrate.NewWithDatabaseInstance, func(string, string, database.Driver) (*migrate.Migrate, error) {
		return &migrate.Migrate{}, nil
	})
	patchUp = monkey.PatchInstanceMethod(reflect.TypeOf(&migrate.Migrate{}), "Up", func(*migrate.Migrate) error {
		if cnt == 0 {
			cnt++
			return migrate.ErrDirty{Version: 1}
		}
		return nil
	})
	patchForce := monkey.PatchInstanceMethod(reflect.TypeOf(&migrate.Migrate{}), "Force", func(_ *migrate.Migrate, v int) error {
		forced = v
		return nil
	})
	patchSteps := monkey.PatchInstanceMethod(reflect.TypeOf(&migrate.Migrate{}), "Steps", func(_ *migrate.Migrate, n int) error {
		steps = n
		return nil
	})
	if err := RunMigrations("file://migrations"); err != nil {
		t.Fatalf("unexpected %v", err)
	}
	if forced != 1 || steps != -1 {
		t.Fatalf("expected force 1 and steps -1, got %d and %d", forced, steps)
	}
	patchSteps.Unpatch()
	patchForce.Unpatch()
	patchUp.Unpatch()
	patchNew.Unpatch()

	// dirty db skip strategy: force but no rollback
	cnt = 0
	forced = 0
	stepsCalled := false
	config.Cfg.Database.DirtyStrategy = enums.DirtyStrategySkip
	patchNew = monkey.Patch(migrate.NewWithDatabaseInstance, func(string, string, database.Driver) (*migrate.Migrate, error) {
		return &migrate.Migrate{}, nil
	})
	patchUp = monkey.PatchInstanceMethod(reflect.TypeOf(&migrate.Migrate{}), "Up", func(*migrate.Migrate) error {
		if cnt == 0 {
			cnt++
			return migrate.ErrDirty{Version: 2}
		}
		return nil
	})
	patchForce = monkey.PatchInstanceMethod(reflect.TypeOf(&migrate.Migrate{}), "Force", func(_ *migrate.Migrate, v int) error {
		forced = v
		return nil
	})
	patchSteps = monkey.PatchInstanceMethod(reflect.TypeOf(&migrate.Migrate{}), "Steps", func(_ *migrate.Migrate, _ int) error {
		stepsCalled = true
		return nil
	})
	if err := RunMigrations("file://migrations"); err != nil {
		t.Fatalf("skip strategy: unexpected %v", err)
	}
	if forced != 2 {
		t.Fatalf("expected force 2, got %d", forced)
	}
	if stepsCalled {
		t.Fatal("skip strategy should not call Steps")
	}
	patchSteps.Unpatch()
	patchForce.Unpatch()
	patchUp.Unpatch()
	patchNew.Unpatch()
	config.Cfg.Database.DirtyStrategy = enums.DirtyStrategyRetry

	// err no change
	patchNew = monkey.Patch(migrate.NewWithDatabaseInstance, func(string, string, database.Driver) (*migrate.Migrate, error) {
		return &migrate.Migrate{}, nil
	})
	patchUp = monkey.PatchInstanceMethod(reflect.TypeOf(&migrate.Migrate{}), "Up", func(*migrate.Migrate) error { return migrate.ErrNoChange })
	patchDriver.Unpatch()
	patchDriver = monkey.Patch(mysqlmigrate.WithInstance, func(*sql.DB, *mysqlmigrate.Config) (database.Driver, error) { return nil, nil })
	if err := RunMigrations("file://migrations"); err != nil {
		t.Fatalf("unexpected %v", err)
	}
	patchUp.Unpatch()
	patchNew.Unpatch()
	patchDriver.Unpatch()
}

type testDB struct{ *sql.DB }

func (t testDB) New() error                          { return nil }
func (testDB) SetLogger(logger.Logger)               {}
func (t testDB) GetDB() *sql.DB                      { return t.DB }
func (t testDB) GetGoquDialect() goqu.DialectWrapper { return goqu.Dialect("mysql") }
func (t testDB) ExecContext(ctx context.Context, q string, args ...interface{}) (sql.Result, error) {
	return t.DB.ExecContext(ctx, q, args...)
}
func (t testDB) WithTransaction(_ context.Context, _ func(ctx context.Context) error) error {
	return nil
}
func (t testDB) BeginTx(_ context.Context, _ *sql.TxOptions) (*sql.Tx, error) { return nil, nil }
func (t testDB) Close() error                                                 { return nil }
func (t testDB) Ping() error                                                  { return nil }
