package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"project-template/infrastructure/config"
	"reflect"
	"strings"
	"sync"
	"testing"

	"project-template/infrastructure/utils"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/bouk/monkey"
)

func stubConfig() {
	config.Cfg = &config.Config{Database: config.DatabaseConfig{User: "u", Password: "p", Host: "h", Port: "0", DBName: "d"}}
}
func TestGetDBInstanceSingleton(t *testing.T) {
	instance = nil
	once = sync.Once{}
	d1 := GetDBInstance()
	d2 := GetDBInstance()
	if d1 != d2 {
		t.Fatalf("expected singleton")
	}
}

func TestExecContextWithTx(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			return
		}
	}(db)

	mock.ExpectBegin()
	tx, _ := db.Begin()

	impl := &dbImpl{db: db}
	ctx := utils.SetTxCtx(context.Background(), tx)

	mock.ExpectExec("INSERT").WithArgs(1).WillReturnResult(sqlmock.NewResult(1, 1))
	if _, err := impl.ExecContext(ctx, "INSERT", 1); err != nil {
		t.Fatalf("ExecContext error: %v", err)
	}
}

func TestExecContextNoTx(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			return
		}
	}(db)

	impl := &dbImpl{db: db}
	ctx := context.Background()

	mock.ExpectExec("UPDATE").WithArgs(2).WillReturnResult(sqlmock.NewResult(1, 1))
	if _, err := impl.ExecContext(ctx, "UPDATE", 2); err != nil {
		t.Fatalf("ExecContext error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPingAndClose(t *testing.T) {
	impl := &dbImpl{}
	if err := impl.Ping(); err == nil {
		t.Fatalf("expected ping error with nil db")
	}
	if err := impl.Close(); err != nil {
		t.Fatalf("close error: %v", err)
	}
}

func TestGetGoquDialect(t *testing.T) {
	impl := &dbImpl{}
	ds := impl.GetGoquDialect().Insert("t")
	if ds.Dialect().Dialect() != "mysql" {
		t.Fatalf("unexpected dialect %s", ds.Dialect().Dialect())
	}
}

func TestWithTransactionCommit(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func(sqlDB *sql.DB) {
		err := sqlDB.Close()
		if err != nil {
			return
		}
	}(sqlDB)
	impl := &dbImpl{db: sqlDB}

	mock.ExpectBegin()
	mock.ExpectCommit()

	err = impl.WithTransaction(context.Background(), func(ctx context.Context) error {
		if utils.GetTxCxt(ctx) == nil {
			t.Fatalf("expected tx in context")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestWithTransactionRollback(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func(sqlDB *sql.DB) {
		err := sqlDB.Close()
		if err != nil {
			return
		}
	}(sqlDB)
	impl := &dbImpl{db: sqlDB}

	mock.ExpectBegin()
	mock.ExpectRollback()

	err = impl.WithTransaction(context.Background(), func(ctx context.Context) error {
		return errors.New("fail")
	})
	if err == nil || err.Error() != "fail" {
		t.Fatalf("expected fail error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestWithTransactionPanic(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func(sqlDB *sql.DB) {
		err := sqlDB.Close()
		if err != nil {
			return
		}
	}(sqlDB)
	impl := &dbImpl{db: sqlDB}

	mock.ExpectBegin()
	mock.ExpectRollback()

	err = impl.WithTransaction(context.Background(), func(ctx context.Context) error {
		panic("boom")
	})
	if err == nil {
		t.Fatal("expected error from panic, got nil")
	}
	if !strings.Contains(err.Error(), "panic in transaction") {
		t.Fatalf("expected panic error, got: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestBeginTx(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func(sqlDB *sql.DB) {
		err := sqlDB.Close()
		if err != nil {
			return
		}
	}(sqlDB)
	impl := &dbImpl{db: sqlDB}

	mock.ExpectBegin()

	tx, err := impl.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tx == nil {
		t.Fatalf("expected tx")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestBeginTxError(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func(sqlDB *sql.DB) {
		err := sqlDB.Close()
		if err != nil {
			return
		}
	}(sqlDB)
	impl := &dbImpl{db: sqlDB}

	mock.ExpectBegin().WillReturnError(sql.ErrConnDone)

	tx, err := impl.BeginTx(context.Background(), nil)
	if tx != nil {
		t.Fatalf("expected nil tx")
	}
	if !errors.Is(err, sql.ErrConnDone) {
		t.Fatalf("expected err %v, got %v", sql.ErrConnDone, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestNewOpenError(t *testing.T) {
	impl := &dbImpl{}
	stubConfig()
	db, _, _ := sqlmock.New()
	monkey.Patch(log.Fatalf, func(string, ...interface{}) { panic("fatal") })
	defer monkey.Unpatch(log.Fatalf)
	monkey.Patch(log.Println, func(...interface{}) {})
	defer monkey.Unpatch(log.Println)
	monkey.Patch(sql.Open, func(string, string) (*sql.DB, error) { return db, errors.New("open") })
	defer monkey.Unpatch(sql.Open)
	monkey.PatchInstanceMethod(reflect.TypeOf(&sql.DB{}), "PingContext", func(*sql.DB, context.Context) error { return nil })
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(&sql.DB{}), "PingContext")
	defer func() {
		if r := recover(); r == nil || r != "fatal" {
			t.Fatalf("expected panic")
		}
	}()
	_ = impl.New()
}

func TestNewPingError(t *testing.T) {
	impl := &dbImpl{}
	stubConfig()
	db, _, _ := sqlmock.New()
	monkey.Patch(log.Fatalf, func(string, ...interface{}) {})
	defer monkey.Unpatch(log.Fatalf)
	monkey.Patch(log.Println, func(...interface{}) {})
	defer monkey.Unpatch(log.Println)
	monkey.Patch(sql.Open, func(string, string) (*sql.DB, error) { return db, nil })
	defer monkey.Unpatch(sql.Open)
	calledClose := false
	monkey.PatchInstanceMethod(reflect.TypeOf(&sql.DB{}), "PingContext", func(*sql.DB, context.Context) error { return errors.New("ping") })
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(&sql.DB{}), "PingContext")
	monkey.PatchInstanceMethod(reflect.TypeOf(db), "Close", func(*sql.DB) error { calledClose = true; return nil })
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(db), "Close")
	if err := impl.New(); err == nil {
		t.Fatalf("expected error")
	}
	if !calledClose {
		t.Fatalf("close not called")
	}
	if impl.db != nil {
		t.Fatalf("db should be nil")
	}
}

func TestNewPingCloseError(t *testing.T) {
	impl := &dbImpl{}
	stubConfig()
	db, _, _ := sqlmock.New()
	monkey.Patch(log.Fatalf, func(string, ...interface{}) { panic("fatal") })
	defer monkey.Unpatch(log.Fatalf)
	monkey.Patch(log.Println, func(...interface{}) {})
	defer monkey.Unpatch(log.Println)
	monkey.Patch(sql.Open, func(string, string) (*sql.DB, error) { return db, nil })
	defer monkey.Unpatch(sql.Open)
	monkey.PatchInstanceMethod(reflect.TypeOf(&sql.DB{}), "PingContext", func(*sql.DB, context.Context) error { return errors.New("ping") })
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(&sql.DB{}), "PingContext")
	monkey.PatchInstanceMethod(reflect.TypeOf(&sql.DB{}), "Close", func(*sql.DB) error { return errors.New("close") })
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(&sql.DB{}), "Close")
	defer func() {
		if r := recover(); r == nil || r != "fatal" {
			t.Fatalf("expected panic")
		}
	}()
	_ = impl.New()
}

func TestNewSuccess(t *testing.T) {
	impl := &dbImpl{}
	stubConfig()
	db, _, _ := sqlmock.New()
	monkey.Patch(log.Fatalf, func(string, ...interface{}) { panic("fatal") })
	defer monkey.Unpatch(log.Fatalf)
	monkey.Patch(log.Println, func(...interface{}) {})
	defer monkey.Unpatch(log.Println)
	monkey.Patch(sql.Open, func(string, string) (*sql.DB, error) { return db, nil })
	defer monkey.Unpatch(sql.Open)
	monkey.PatchInstanceMethod(reflect.TypeOf(&sql.DB{}), "PingContext", func(*sql.DB, context.Context) error { return nil })
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(&sql.DB{}), "PingContext")
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unexpected panic: %v", r)
		}
	}()
	if err := impl.New(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if impl.db != db {
		t.Fatalf("db not set")
	}
}

func TestGetDBPanics(t *testing.T) {
	impl := &dbImpl{}
	monkey.Patch(log.Fatalf, func(string, ...interface{}) { panic("fatal") })
	defer monkey.Unpatch(log.Fatalf)
	defer func() {
		if r := recover(); r == nil || !strings.Contains(r.(string), "fatal") {
			t.Fatalf("expected panic")
		}
	}()
	_ = impl.GetDB()
}

func TestGetDB(t *testing.T) {
	db, _, _ := sqlmock.New()
	impl := &dbImpl{db: db}
	if got := impl.GetDB(); got != db {
		t.Fatalf("got wrong db")
	}
}

func TestPingAndCloseWithDB(t *testing.T) {
	db, _, _ := sqlmock.New()
	impl := &dbImpl{db: db}
	calledPing := false
	calledClose := false
	monkey.PatchInstanceMethod(reflect.TypeOf(&sql.DB{}), "PingContext", func(*sql.DB, context.Context) error { calledPing = true; return nil })
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(&sql.DB{}), "PingContext")
	monkey.PatchInstanceMethod(reflect.TypeOf(db), "Close", func(*sql.DB) error { calledClose = true; return nil })
	defer monkey.UnpatchInstanceMethod(reflect.TypeOf(db), "Close")
	if err := impl.Ping(); err != nil || !calledPing {
		t.Fatalf("ping failed %v %v", err, calledPing)
	}
	if err := impl.Close(); err != nil || !calledClose {
		t.Fatalf("close failed %v %v", err, calledClose)
	}
}

func TestLogErrorWithoutLogger(t *testing.T) {
	// Exercise the else branch of logError (no logger set, falls back to stdlog).
	impl := &dbImpl{}
	var buf strings.Builder
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)
	impl.logError("test error: %v", "detail")
	if !strings.Contains(buf.String(), "test error: detail") {
		t.Fatalf("expected log output, got %q", buf.String())
	}
}

func TestLogInfoWithoutLogger(t *testing.T) {
	impl := &dbImpl{}
	var buf strings.Builder
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)
	impl.logInfo("info msg: %s", "hello")
	if !strings.Contains(buf.String(), "info msg: hello") {
		t.Fatalf("expected log output, got %q", buf.String())
	}
}

type errorCapLogger struct {
	errorMsgs []string
}

func (e *errorCapLogger) Debug(string)                  {}
func (e *errorCapLogger) DebugF(string, ...interface{}) {}
func (e *errorCapLogger) Info(string)                    {}
func (e *errorCapLogger) InfoF(string, ...interface{})  {}
func (e *errorCapLogger) Warn(string)                    {}
func (e *errorCapLogger) WarnF(string, ...interface{})  {}
func (e *errorCapLogger) Error(string)                   {}
func (e *errorCapLogger) ErrorF(format string, args ...interface{}) {
	e.errorMsgs = append(e.errorMsgs, fmt.Sprintf(format, args...))
}
func (e *errorCapLogger) Fatal(string)                   {}
func (e *errorCapLogger) FatalF(string, ...interface{}) {}
func (e *errorCapLogger) ParentID() string               { return "" }
func (e *errorCapLogger) ChildID() string                { return "" }

func TestLogErrorWithLogger(t *testing.T) {
	l := &errorCapLogger{}
	impl := &dbImpl{logger: l}
	impl.logError("something went wrong: %v", "oops")
	if len(l.errorMsgs) != 1 || !strings.Contains(l.errorMsgs[0], "something went wrong: oops") {
		t.Fatalf("expected logger.ErrorF to be called, got %v", l.errorMsgs)
	}
}

func TestWithTransactionErrors(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func(sqlDB *sql.DB) {
		err := sqlDB.Close()
		if err != nil {
			return
		}
	}(sqlDB)
	impl := &dbImpl{db: sqlDB}

	// begin error
	mock.ExpectBegin().WillReturnError(sql.ErrConnDone)
	if err := impl.WithTransaction(context.Background(), func(ctx context.Context) error { return nil }); err == nil {
		t.Fatalf("expected begin error")
	}

	// commit error
	mock.ExpectBegin()
	mock.ExpectCommit().WillReturnError(sql.ErrTxDone)
	if err := impl.WithTransaction(context.Background(), func(ctx context.Context) error { return nil }); err == nil {
		t.Fatalf("expected commit error")
	}

	// rollback error
	mock.ExpectBegin()
	mock.ExpectRollback().WillReturnError(errors.New("rb"))
	if err := impl.WithTransaction(context.Background(), func(ctx context.Context) error { return errors.New("f") }); err == nil || !strings.Contains(err.Error(), "rb") {
		t.Fatalf("expected rollback error")
	}

	// panic rollback error is logged, but the panic itself is returned as an error
	mock.ExpectBegin()
	mock.ExpectRollback().WillReturnError(errors.New("prb"))
	if err := impl.WithTransaction(context.Background(), func(ctx context.Context) error { panic("boom") }); err == nil || !strings.Contains(err.Error(), "panic in transaction") {
		t.Fatalf("expected panic error, got: %v", err)
	}
}
