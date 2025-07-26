package repositories

import (
	"context"
	"database/sql"
	"testing"

	"github.com/doug-martin/goqu/v9"

	"project-template/infrastructure/utils"
	"project-template/pkg/logger"
)

type mockDB struct{ db *sql.DB }

func (m mockDB) New() error                          { return nil }
func (mockDB) SetLogger(logger.Logger)               {}
func (m mockDB) GetDB() *sql.DB                      { return m.db }
func (m mockDB) GetGoquDialect() goqu.DialectWrapper { return goqu.Dialect("mysql") }
func (m mockDB) ExecContext(ctx context.Context, q string, args ...interface{}) (sql.Result, error) {
	return m.db.ExecContext(ctx, q, args...)
}
func (m mockDB) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
		}
	}()
	if err = fn(utils.SetTxCtx(ctx, tx)); err != nil {
		if rb := tx.Rollback(); rb != nil {
			return rb
		}
		return err
	}
	return tx.Commit()
}
func (m mockDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return m.db.BeginTx(ctx, opts)
}
func (m mockDB) Close() error { return m.db.Close() }
func (m mockDB) Ping() error  { return nil }

type dummyLogger struct{}

func (dummyLogger) DebugF(string, ...interface{}) {}
func (dummyLogger) InfoF(string, ...interface{})  {}
func (dummyLogger) WarnF(string, ...interface{})  {}
func (dummyLogger) ErrorF(string, ...interface{}) {}
func (dummyLogger) FatalF(string, ...interface{}) {}
func (dummyLogger) ParentID() string              { return "" }
func (dummyLogger) ChildID() string               { return "" }
func (dummyLogger) CloseLogFile()                 {}

func TestRepositoryCaching(t *testing.T) {
	repo := NewRepository(dummyLogger{}, mockDB{}).(*Repository)
	i1 := repo.GetItemRepository()
	i2 := repo.GetItemRepository()
	if i1 != i2 {
		t.Fatalf("item repo not cached")
	}
}
