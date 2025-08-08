package testutil

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"

	"project-template/infrastructure/config"
	"project-template/infrastructure/db"
	"project-template/pkg/logger"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/bouk/monkey"
)

func TestNewDBMock(t *testing.T) {
	dbMock, mock := NewDBMock(t)
	if dbMock == nil || mock == nil {
		t.Fatalf("nil")
	}
	mock.ExpectPing()
	if err := dbMock.Ping(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPatchDBAndLogger(t *testing.T) {
	sqlDB, _, _ := sqlmock.New()
	patch := PatchDB(sqlDB)
	defer patch.Unpatch()
	if db.GetDBInstance().GetDB() != sqlDB {
		t.Fatalf("not patched")
	}

	p := PatchLogger()
	defer p.Unpatch()
	l, err := logger.NewLogger(config.LogConfig{Path: t.TempDir(), FileName: "a"}, "p", "c")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := l.(StubLogger); !ok {
		t.Fatalf("wrong type")
	}
}

func TestWriteKeyFiles(t *testing.T) {
	dir := t.TempDir()
	priv, pub := WriteKeyFiles(t, dir)
	if _, err := os.Stat(priv); err != nil {
		t.Fatalf("priv: %v", err)
	}
	if _, err := os.Stat(pub); err != nil {
		t.Fatalf("pub: %v", err)
	}
}

func TestMockDBWithTransaction(t *testing.T) {
	sqlDB, mock, _ := sqlmock.New()
	m := MockDB{DB: sqlDB}

	mock.ExpectBegin()
	mock.ExpectCommit()
	if err := m.WithTransaction(context.Background(), func(context.Context) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}

	mock.ExpectBegin()
	mock.ExpectRollback()
	expErr := errors.New("fail")
	if err := m.WithTransaction(context.Background(), func(context.Context) error { return expErr }); err != expErr {
		t.Fatalf("%v != %v", err, expErr)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestStubLoggerAndMockDBMethods(t *testing.T) {
	var l StubLogger
	l.DebugF("d")
	l.InfoF("i")
	l.WarnF("w")
	l.ErrorF("e")
	l.FatalF("f")
	if l.ParentID() != "p" || l.ChildID() != "c" {
		t.Fatalf("ids")
	}
	l.CloseLogFile()

	sqlDB, mock, _ := sqlmock.New()
	m := MockDB{DB: sqlDB}
	if err := m.New(); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(m.GetGoquDialect()) != "{mysql}" {
		t.Fatalf("dialect")
	}
	mock.ExpectExec("SELECT 1").WillReturnResult(sqlmock.NewResult(0, 0))
	if _, err := m.ExecContext(context.Background(), "SELECT 1"); err != nil {
		t.Fatal(err)
	}
	mock.ExpectBegin()
	tx, err := m.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectRollback()
	_ = tx.Rollback()
	mock.ExpectClose()
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	if m.Ping() != nil {
		t.Fatalf("ping")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestWithTransactionBeginError(t *testing.T) {
	sqlDB, _, _ := sqlmock.New()
	m := MockDB{DB: sqlDB}
	patch := monkey.PatchInstanceMethod(reflect.TypeOf(sqlDB), "BeginTx", func(*sql.DB, context.Context, *sql.TxOptions) (*sql.Tx, error) {
		return nil, errors.New("begin")
	})
	defer patch.Unpatch()
	if err := m.WithTransaction(context.Background(), func(ctx context.Context) error { return nil }); err == nil || err.Error() != "begin" {
		t.Fatalf("%v", err)
	}
}

func TestMockDBSetLoggerNoop(t *testing.T) {
	sqlDB, _, _ := sqlmock.New()
	m := MockDB{DB: sqlDB}
	m.SetLogger(StubLogger{})
	if m.DB != sqlDB {
		t.Fatalf("SetLogger altered DB")
	}
}
