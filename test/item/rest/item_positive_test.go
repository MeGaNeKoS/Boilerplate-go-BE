//go:build !grpc && !kafka

package item_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"unsafe"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/bouk/monkey"

	"project-template/cmd"
	"project-template/infrastructure/config"
	models "project-template/infrastructure/dto/item"
	"project-template/infrastructure/dto/response"
	"project-template/infrastructure/utils"
	"project-template/outbound/transport"
	codepkg "project-template/pkg/code"
	"project-template/pkg/logger"
	"project-template/test/testutil"
)

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("boom") }
func (errReader) Close() error             { return nil }

// notFoundItemRepo implements repoitem.Repository returning ErrItemNotFound
// for update and delete operations.
type notFoundItemRepo struct{}

func (notFoundItemRepo) Create(context.Context, *models.Item) error     { return nil }
func (notFoundItemRepo) List(context.Context) ([]models.Item, error)    { return nil, nil }
func (notFoundItemRepo) Get(context.Context, int) (*models.Item, error) { return nil, nil }
func (notFoundItemRepo) Update(context.Context, *models.Item) error     { return codepkg.ErrItemNotFound }
func (notFoundItemRepo) Delete(context.Context, int) error              { return codepkg.ErrItemNotFound }

func setupServer(t *testing.T) (*http.Server, sqlmock.Sqlmock, string) {
	t.Helper()
	dir := t.TempDir()
	priv, pub := testutil.WriteKeyFiles(t, dir)

	config.Cfg = &config.Config{
		AppName: "APP",
		Server: config.ServerConfig{
			Environment:   "dev",
			SkipTLSVerify: true,
			Endpoint:      config.EndpointConfig{Based: "/api"},
			Timeout:       config.TimeoutConfig{Server: 1, Read: 1, Write: 1, Idle: 1},
			JWT:           config.JWTConfig{PrivateKey: priv, PublicKey: pub},
		},
		REST:      config.ListenerConfig{Host: "127.0.0.1", Port: "0"},
		LogTarget: config.LogConfig{Path: dir, FileName: "app.log"},
		Service:   config.Service{Example: config.ServiceDetail{Host: "http://example"}},
	}

	if err := utils.InitializeJWTService(true, true); err != nil {
		t.Fatal(err)
	}
	tok, _ := utils.GetJWTService().SignJWT(nil)

	sqlDB, mock := testutil.NewDBMock(t)
	patchDB := testutil.PatchDB(sqlDB)
	t.Cleanup(patchDB.Unpatch)

	patchLogger := testutil.PatchLogger()
	t.Cleanup(patchLogger.Unpatch)

	patchHTTP := monkey.PatchInstanceMethod(reflect.TypeOf(&transport.HTTPOutbound{}), "SendHTTPRequest", func(_ *transport.HTTPOutbound, _ logger.Logger) (response.HttpResponse, *codepkg.Code) {
		gr := &response.GenericResponse[any]{}
		gr.Body = models.Item{ID: 5, Name: "x"}
		return response.HttpResponse{HTTPCode: http.StatusOK, RawResponsePayload: gr}, nil
	})
	t.Cleanup(patchHTTP.Unpatch)

	srv := cmd.GetRESTServer(config.Cfg, testutil.StubLogger{})
	sv := reflect.ValueOf(srv).Elem().FieldByName("httpServer")
	httpSrv := reflect.NewAt(sv.Type(), unsafe.Pointer(sv.UnsafeAddr())).Elem().Interface().(*http.Server)
	return httpSrv, mock, tok
}

func TestListItemsIntegration(t *testing.T) {
	httpSrv, mock, tok := setupServer(t)

	rows := sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "foo")
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/api/items", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()

	httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("code %d", rec.Code)
	}

	var resp response.GenericResponse[[]models.Item]
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	items := resp.Body
	if len(items) != 1 || items[0].ID != 1 {
		t.Fatalf("unexpected items %#v", items)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
func TestCreateItemIntegration(t *testing.T) {
	httpSrv, mock, tok := setupServer(t)

	mock.ExpectBegin()
	mock.ExpectExec("INSERT").WillReturnResult(sqlmock.NewResult(3, 1))
	mock.ExpectCommit()

	req := httptest.NewRequest(http.MethodPost, "/api/items", bytes.NewBufferString(`{"id":0,"name":"foo"}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("code %d", rec.Code)
	}

	var resp response.GenericResponse[models.Item]
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	item := resp.Body
	if item.ID != 3 || item.Name != "foo" {
		t.Fatalf("unexpected item %#v", item)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGetItemIntegration(t *testing.T) {
	httpSrv, mock, tok := setupServer(t)

	rows := sqlmock.NewRows([]string{"id", "name"}).AddRow(2, "bar")
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/api/items/2", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()

	httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("code %d", rec.Code)
	}

	var resp response.GenericResponse[models.Item]
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	item := resp.Body
	if item.ID != 2 || item.Name != "bar" {
		t.Fatalf("unexpected item %#v", item)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateItemIntegration(t *testing.T) {
	httpSrv, mock, tok := setupServer(t)

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	req := httptest.NewRequest(http.MethodPut, "/api/items/4", bytes.NewBufferString(`{"id":4,"name":"baz"}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("code %d", rec.Code)
	}

	var resp response.GenericResponse[models.Item]
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	item := resp.Body
	if item.ID != 4 || item.Name != "baz" {
		t.Fatalf("unexpected item %#v", item)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteItemIntegration(t *testing.T) {
	httpSrv, mock, tok := setupServer(t)

	mock.ExpectBegin()
	mock.ExpectExec("DELETE").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	req := httptest.NewRequest(http.MethodDelete, "/api/items/5", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()

	httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("code %d", rec.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
