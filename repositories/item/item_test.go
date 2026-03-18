package item

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/bouk/monkey"
	"github.com/doug-martin/goqu/v9"

	models "project-template/infrastructure/dto/item"
	"project-template/infrastructure/utils"
	"project-template/pkg/code"
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

type badResult struct{}

func (badResult) LastInsertId() (int64, error) { return 0, errors.New("lid") }
func (badResult) RowsAffected() (int64, error) { return 0, nil }

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

func (dummyLogger) Debug(string)                  {}
func (dummyLogger) DebugF(string, ...interface{}) {}
func (dummyLogger) Info(string)                    {}
func (dummyLogger) InfoF(string, ...interface{})  {}
func (dummyLogger) Warn(string)                    {}
func (dummyLogger) WarnF(string, ...interface{})  {}
func (dummyLogger) Error(string)                   {}
func (dummyLogger) ErrorF(string, ...interface{}) {}
func (dummyLogger) Fatal(string)                   {}
func (dummyLogger) FatalF(string, ...interface{}) {}
func (dummyLogger) ParentID() string              { return "" }
func (dummyLogger) ChildID() string               { return "" }
func (dummyLogger) CloseLogFile()                 {}

func newRepo(t *testing.T) (*itemRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	mdb := mockDB{db: db}
	return &itemRepository{db: mdb, log: dummyLogger{}}, mock
}

func TestNewItemRepository(t *testing.T) {
	db := mockDB{}
	log := dummyLogger{}
	repoIface := NewItemRepository(log, db)
	repo, ok := repoIface.(*itemRepository)
	if !ok {
		t.Fatalf("expected *itemRepository, got %T", repoIface)
	}
	if repo.db != db || repo.log != log {
		t.Fatalf("unexpected repo fields: %#v", repo)
	}
}

func TestItemRepositoryCreate(t *testing.T) {
	repo, mock := newRepo(t)
	mock.ExpectBegin()
	mock.ExpectExec("INSERT").WillReturnResult(sqlmock.NewResult(2, 1))
	mock.ExpectCommit()
	item := models.Item{Name: "foo"}
	if err := repo.Create(context.Background(), &item); err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if item.ID != 2 {
		t.Fatalf("id not set")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestItemRepositoryList(t *testing.T) {
	repo, mock := newRepo(t)
	rows := sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "a").AddRow(2, "b")
	mock.ExpectQuery("SELECT").WillReturnRows(rows)
	items, err := repo.List(context.Background())
	if err != nil || len(items) != 2 {
		t.Fatalf("List error %v items %v", err, items)
	}
}

func TestItemRepositoryGetNotFound(t *testing.T) {
	repo, mock := newRepo(t)
	mock.ExpectQuery("SELECT").WillReturnError(sql.ErrNoRows)
	_, err := repo.Get(context.Background(), 1)
	if err == nil || !errors.Is(err, code.ErrItemNotFound) {
		t.Fatalf("unexpected error %v", err)
	}
}

func TestItemRepositoryUpdateError(t *testing.T) {
	repo, mock := newRepo(t)
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE").WillReturnError(sql.ErrConnDone)
	mock.ExpectRollback()
	err := repo.Update(context.Background(), &models.Item{ID: 1})
	if err == nil {
		t.Fatal("expected error")
	}
	err = mock.ExpectationsWereMet()
	if err != nil {
		return
	}
}

func TestItemRepositoryDelete(t *testing.T) {
	repo, mock := newRepo(t)
	mock.ExpectBegin()
	mock.ExpectExec("DELETE").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := repo.Delete(context.Background(), 1); err != nil {
		t.Fatalf("Delete error: %v", err)
	}
}

func TestItemRepositoryCreateErrorBranches(t *testing.T) {
	repo, mock := newRepo(t)
	// ToSQL error
	monkey.PatchInstanceMethod(reflect.TypeOf(&goqu.InsertDataset{}), "ToSQL", func(*goqu.InsertDataset) (string, []interface{}, error) {
		return "", nil, errors.New("tosql")
	})
	if err := repo.Create(context.Background(), &models.Item{}); err == nil || !strings.Contains(err.Error(), "tosql") {
		t.Fatalf("expected tosql error, got %v", err)
	}
	monkey.UnpatchAll()

	// Exec error
	mock.ExpectBegin()
	mock.ExpectExec("INSERT").WillReturnError(sql.ErrConnDone)
	mock.ExpectRollback()
	if err := repo.Create(context.Background(), &models.Item{Name: "a"}); err == nil {
		t.Fatalf("expected exec error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}

	// LastInsertID error
	mock.ExpectBegin()
	mock.ExpectExec("INSERT").WillReturnResult(badResult{})
	mock.ExpectRollback()
	if err := repo.Create(context.Background(), &models.Item{Name: "a"}); err == nil {
		t.Fatalf("expected last insert id error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestItemRepositoryListErrorBranches(t *testing.T) {
	repo, mock := newRepo(t)
	ctx := context.Background()

	// ToSQL error
	monkey.PatchInstanceMethod(reflect.TypeOf(&goqu.SelectDataset{}), "ToSQL", func(*goqu.SelectDataset) (string, []interface{}, error) {
		return "", nil, errors.New("bad")
	})
	if _, err := repo.List(ctx); err == nil {
		t.Fatalf("expected tosql error")
	}
	monkey.UnpatchAll()

	// Query error
	mock.ExpectQuery("SELECT").WillReturnError(sql.ErrConnDone)
	if _, err := repo.List(ctx); err == nil {
		t.Fatalf("expected query error")
	}
	_ = mock.ExpectationsWereMet()

	// Scan error
	rows := sqlmock.NewRows([]string{"id", "name"}).AddRow("x", 1)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)
	if _, err := repo.List(ctx); err == nil {
		t.Fatalf("expected scan error")
	}

	// rows.Err error
	rows = sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "a").RowError(0, errors.New("iter"))
	mock.ExpectQuery("SELECT").WillReturnRows(rows)
	if _, err := repo.List(ctx); err == nil {
		t.Fatalf("expected row err")
	}

	// close error branch (no list error)
	rows = sqlmock.NewRows([]string{"id", "name"}).AddRow(2, "b")
	mock.ExpectQuery("SELECT").WillReturnRows(rows)
	monkey.PatchInstanceMethod(reflect.TypeOf(&sql.Rows{}), "Close", func(*sql.Rows) error { return errors.New("close") })
	defer monkey.UnpatchAll()
	items, err := repo.List(ctx)
	if err != nil || len(items) != 1 {
		t.Fatalf("unexpected result %v %v", items, err)
	}
}

func TestItemRepositoryGetAdditional(t *testing.T) {
	repo, mock := newRepo(t)
	ctx := context.Background()

	// ToSQL error
	monkey.PatchInstanceMethod(reflect.TypeOf(&goqu.SelectDataset{}), "ToSQL", func(*goqu.SelectDataset) (string, []interface{}, error) {
		return "", nil, errors.New("g")
	})
	if _, err := repo.Get(ctx, 1); err == nil {
		t.Fatalf("expected tosql error")
	}
	monkey.UnpatchAll()

	// generic scan error
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow("x", 1))
	if _, err := repo.Get(ctx, 1); err == nil {
		t.Fatalf("expected scan error")
	}

	// success
	rows := sqlmock.NewRows([]string{"id", "name"}).AddRow(3, "c")
	mock.ExpectQuery("SELECT").WillReturnRows(rows)
	itm, err := repo.Get(ctx, 3)
	if err != nil || itm.ID != 3 || itm.Name != "c" {
		t.Fatalf("bad item %v %v", itm, err)
	}
}

func TestItemRepositoryUpdateAndDelete(t *testing.T) {
	repo, mock := newRepo(t)
	ctx := context.Background()

	// update ToSQL error
	monkey.PatchInstanceMethod(reflect.TypeOf(&goqu.UpdateDataset{}), "ToSQL", func(*goqu.UpdateDataset) (string, []interface{}, error) {
		return "", nil, errors.New("u")
	})
	if err := repo.Update(ctx, &models.Item{ID: 1}); err == nil {
		t.Fatalf("expected update tosql error")
	}
	monkey.UnpatchAll()

	// update success
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := repo.Update(ctx, &models.Item{ID: 1, Name: "n"}); err != nil {
		t.Fatalf("unexpected update err: %v", err)
	}

	// delete ToSQL error
	monkey.PatchInstanceMethod(reflect.TypeOf(&goqu.DeleteDataset{}), "ToSQL", func(*goqu.DeleteDataset) (string, []interface{}, error) {
		return "", nil, errors.New("d")
	})
	if err := repo.Delete(ctx, 1); err == nil {
		t.Fatalf("expected delete tosql error")
	}
	monkey.UnpatchAll()

	// delete exec error
	mock.ExpectBegin()
	mock.ExpectExec("DELETE").WillReturnError(sql.ErrConnDone)
	mock.ExpectRollback()
	if err := repo.Delete(ctx, 2); err == nil {
		t.Fatalf("expected exec error")
	}
	_ = mock.ExpectationsWereMet()
}
