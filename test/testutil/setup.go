package testutil

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/doug-martin/goqu/v9"

	"project-template/infrastructure/config"
	"project-template/infrastructure/db"
	"project-template/pkg/logger"

	"github.com/bouk/monkey"
)

// StubLogger implements logger.Logger with no-op methods.
type StubLogger struct{}

func (StubLogger) Debug(_ string)                    {}
func (StubLogger) DebugF(_ string, _ ...interface{}) {}
func (StubLogger) Info(_ string)                     {}
func (StubLogger) InfoF(_ string, _ ...interface{})  {}
func (StubLogger) Warn(_ string)                     {}
func (StubLogger) WarnF(_ string, _ ...interface{})  {}
func (StubLogger) Error(_ string)                    {}
func (StubLogger) ErrorF(_ string, _ ...interface{}) {}
func (StubLogger) Fatal(_ string)                    {}
func (StubLogger) FatalF(_ string, _ ...interface{}) {}
func (StubLogger) ParentID() string                  { return "p" }
func (StubLogger) ChildID() string                   { return "c" }
func (StubLogger) CloseLogFile()                     {}

// MockDB implements the db.DB interface for integration tests.
type MockDB struct{ DB *sql.DB }

func (m MockDB) New() error { return nil }

// SetLogger implements db.DB but performs no logging.
func (MockDB) SetLogger(_ logger.Logger)             {}
func (m MockDB) GetDB() *sql.DB                      { return m.DB }
func (m MockDB) GetGoquDialect() goqu.DialectWrapper { return goqu.Dialect("mysql") }
func (m MockDB) ExecContext(ctx context.Context, q string, args ...interface{}) (sql.Result, error) {
	return m.DB.ExecContext(ctx, q, args...)
}
func (m MockDB) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := m.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err = fn(ctx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
func (m MockDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return m.DB.BeginTx(ctx, opts)
}
func (m MockDB) Close() error { return m.DB.Close() }
func (m MockDB) Ping() error  { return nil }

// NewDBMock creates a sql.DB and sqlmock instance for tests.
func NewDBMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	dbMock, mock, _ := sqlmock.New()
	return dbMock, mock
}

// PatchDB replaces db.GetDBInstance with a mock implementation.
func PatchDB(sqlDB *sql.DB) *monkey.PatchGuard {
	return monkey.Patch(db.GetDBInstance, func() db.DB { return MockDB{DB: sqlDB} })
}

// PatchLogger replaces logger.NewLogger with a stub implementation.
func PatchLogger() *monkey.PatchGuard {
	return monkey.Patch(logger.NewLogger, func(cfg config.LogConfig, p, c string) (logger.Logger, error) {
		return StubLogger{}, nil
	})
}

// WriteKeyFiles creates a private/public key pair for JWT tests.
func WriteKeyFiles(t *testing.T, dir string) (string, string) {
	t.Helper()
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	privPath := filepath.Join(dir, "priv.pem")
	pubPath := filepath.Join(dir, "pub.pem")

	privBytes := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	_ = os.WriteFile(privPath, privBytes, 0600)
	pubDer, _ := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	pubBytes := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDer})
	_ = os.WriteFile(pubPath, pubBytes, 0600)
	return privPath, pubPath
}
