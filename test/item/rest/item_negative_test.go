//go:build !grpc && !kafka

package item_test

import (
	"bytes"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/bouk/monkey"

	repoagg "project-template/repositories"
	repoitem "project-template/repositories/item"
)

func TestListItemsIntegrationDBError(t *testing.T) {
	httpSrv, mock, tok := setupServer(t)

	mock.ExpectQuery("SELECT").WillReturnError(sql.ErrConnDone)

	req := httptest.NewRequest(http.MethodGet, "/api/items/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()

	httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("code %d", rec.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCreateItemIntegrationBadJSON(t *testing.T) {
	httpSrv, _, tok := setupServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/items/", bytes.NewBufferString("{"))
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()

	httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestCreateItemIntegrationDBError(t *testing.T) {
	httpSrv, mock, tok := setupServer(t)

	mock.ExpectBegin()
	mock.ExpectExec("INSERT").WillReturnError(sql.ErrConnDone)
	mock.ExpectRollback()

	req := httptest.NewRequest(http.MethodPost, "/api/items/", bytes.NewBufferString(`{"name":"foo"}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()

	httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("code %d", rec.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGetItemIntegrationNotFound(t *testing.T) {
	httpSrv, mock, tok := setupServer(t)

	mock.ExpectQuery("SELECT").WillReturnError(sql.ErrNoRows)

	req := httptest.NewRequest(http.MethodGet, "/api/items/2", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()

	httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("code %d", rec.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateItemIntegrationRepoError(t *testing.T) {
	httpSrv, mock, tok := setupServer(t)

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE").WillReturnError(sql.ErrConnDone)
	mock.ExpectRollback()

	req := httptest.NewRequest(http.MethodPut, "/api/items/4", bytes.NewBufferString(`{"name":"baz"}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()

	httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("code %d", rec.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteItemIntegrationRepoError(t *testing.T) {
	httpSrv, mock, tok := setupServer(t)

	mock.ExpectBegin()
	mock.ExpectExec("DELETE").WillReturnError(sql.ErrConnDone)
	mock.ExpectRollback()

	req := httptest.NewRequest(http.MethodDelete, "/api/items/5", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()

	httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("code %d", rec.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestListItemsIntegrationUnauthorized(t *testing.T) {
	httpSrv, _, _ := setupServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/items/", nil)
	rec := httptest.NewRecorder()

	httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestCreateItemIntegrationUnauthorized(t *testing.T) {
	httpSrv, _, _ := setupServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/items/", bytes.NewBufferString(`{"name":"x"}`))
	rec := httptest.NewRecorder()

	httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestGetItemIntegrationUnauthorized(t *testing.T) {
	httpSrv, _, _ := setupServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/items/1", nil)
	rec := httptest.NewRecorder()

	httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestUpdateItemIntegrationUnauthorized(t *testing.T) {
	httpSrv, _, _ := setupServer(t)

	req := httptest.NewRequest(http.MethodPut, "/api/items/1", bytes.NewBufferString(`{"name":"x"}`))
	rec := httptest.NewRecorder()

	httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestDeleteItemIntegrationUnauthorized(t *testing.T) {
	httpSrv, _, _ := setupServer(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/items/1", nil)
	rec := httptest.NewRecorder()

	httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestCreateItemIntegrationBadBody(t *testing.T) {
	httpSrv, _, tok := setupServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/items/", errReader{})
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()

	httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestUpdateItemIntegrationBadBody(t *testing.T) {
	httpSrv, _, tok := setupServer(t)

	req := httptest.NewRequest(http.MethodPut, "/api/items/1", errReader{})
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()

	httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestUpdateItemIntegrationBadJSON(t *testing.T) {
	httpSrv, _, tok := setupServer(t)

	req := httptest.NewRequest(http.MethodPut, "/api/items/1", bytes.NewBufferString("{"))
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()

	httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestGetItemIntegrationDBError(t *testing.T) {
	httpSrv, mock, tok := setupServer(t)

	mock.ExpectQuery("SELECT").WillReturnError(sql.ErrConnDone)

	req := httptest.NewRequest(http.MethodGet, "/api/items/1", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()

	httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("code %d", rec.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestListItemsIntegrationInvalidToken(t *testing.T) {
	httpSrv, _, _ := setupServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/items/", nil)
	req.Header.Set("Authorization", "Bearer bad")
	rec := httptest.NewRecorder()

	httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestCreateItemIntegrationInvalidToken(t *testing.T) {
	httpSrv, _, _ := setupServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/items/", bytes.NewBufferString(`{"name":"x"}`))
	req.Header.Set("Authorization", "Bearer bad")
	rec := httptest.NewRecorder()

	httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestGetItemIntegrationInvalidToken(t *testing.T) {
	httpSrv, _, _ := setupServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/items/1", nil)
	req.Header.Set("Authorization", "Bearer bad")
	rec := httptest.NewRecorder()

	httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestUpdateItemIntegrationInvalidToken(t *testing.T) {
	httpSrv, _, _ := setupServer(t)

	req := httptest.NewRequest(http.MethodPut, "/api/items/1", bytes.NewBufferString(`{"name":"x"}`))
	req.Header.Set("Authorization", "Bearer bad")
	rec := httptest.NewRecorder()

	httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestDeleteItemIntegrationInvalidToken(t *testing.T) {
	httpSrv, _, _ := setupServer(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/items/1", nil)
	req.Header.Set("Authorization", "Bearer bad")
	rec := httptest.NewRecorder()

	httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code %d", rec.Code)
	}
}

// Bad ID returns 404 because the routes only match numeric IDs.
func TestGetItemIntegrationBadID(t *testing.T) {
	httpSrv, _, tok := setupServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/items/bad", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()

	httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("code %d", rec.Code)
	}
}

// Invalid item ID should not match the route and results in 404.
func TestUpdateItemIntegrationBadID(t *testing.T) {
	httpSrv, _, tok := setupServer(t)

	req := httptest.NewRequest(http.MethodPut, "/api/items/bad", bytes.NewBufferString(`{"name":"x"}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()

	httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("code %d", rec.Code)
	}
}

// Delete with a malformed ID should also produce a 404.
func TestDeleteItemIntegrationBadID(t *testing.T) {
	httpSrv, _, tok := setupServer(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/items/bad", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()

	httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestUpdateItemIntegrationNotFound(t *testing.T) {
	httpSrv, _, tok := setupServer(t)

	patchRepo := monkey.PatchInstanceMethod(reflect.TypeOf(&repoagg.Repository{}), "GetItemRepository", func(*repoagg.Repository) repoitem.Repository {
		return notFoundItemRepo{}
	})
	defer patchRepo.Unpatch()

	req := httptest.NewRequest(http.MethodPut, "/api/items/4", bytes.NewBufferString(`{"name":"baz"}`))
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()

	httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestDeleteItemIntegrationNotFound(t *testing.T) {
	httpSrv, _, tok := setupServer(t)

	patchRepo := monkey.PatchInstanceMethod(reflect.TypeOf(&repoagg.Repository{}), "GetItemRepository", func(*repoagg.Repository) repoitem.Repository {
		return notFoundItemRepo{}
	})
	defer patchRepo.Unpatch()

	req := httptest.NewRequest(http.MethodDelete, "/api/items/9", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()

	httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("code %d", rec.Code)
	}
}
