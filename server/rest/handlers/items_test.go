package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bouk/monkey"
	"github.com/go-chi/chi/v5"

	"project-template/infrastructure/config"
	models "project-template/infrastructure/dto/item"
	"project-template/infrastructure/dto/response"
	"project-template/infrastructure/utils"
	"project-template/outbound/service/example"
	"project-template/pkg/code"
       repoitem "project-template/repositories/item"
       services "project-template/services/item"
)

// --- stubs ---
type stubLogger struct{}

func (stubLogger) DebugF(string, ...interface{}) {}
func (stubLogger) InfoF(string, ...interface{})  {}
func (stubLogger) WarnF(string, ...interface{})  {}
func (stubLogger) ErrorF(string, ...interface{}) {}
func (stubLogger) FatalF(string, ...interface{}) {}
func (stubLogger) ParentID() string              { return "p" }
func (stubLogger) ChildID() string               { return "c" }
func (stubLogger) CloseLogFile()                 {}

type stubItemRepo struct{}

func (stubItemRepo) Create(context.Context, *models.Item) error     { return nil }
func (stubItemRepo) List(context.Context) ([]models.Item, error)    { return nil, nil }
func (stubItemRepo) Get(context.Context, int) (*models.Item, error) { return nil, nil }
func (stubItemRepo) Update(context.Context, *models.Item) error     { return nil }
func (stubItemRepo) Delete(context.Context, int) error              { return nil }

type stubRepo struct{}

func (stubRepo) GetItemRepository() repoitem.Repository { return stubItemRepo{} }

type stubExampleOutbound struct{}

func (stubExampleOutbound) FetchItemByID(_ context.Context, id int) (models.Item, error) {
	return models.Item{ID: id}, nil
}
func (stubExampleOutbound) FetchItemByFilter(_ context.Context, filter string) ([]models.Item, error) {
	return nil, nil
}

type stubExampleAgg struct{}

func (stubExampleAgg) HTTP() example.ExampleOutbound       { return stubExampleOutbound{} }
func (stubExampleAgg) GRPC() example.ExampleGRPCOutbound   { return nil }
func (stubExampleAgg) Kafka() example.ExampleKafkaOutbound { return nil }

type stubOutbound struct{}

func (stubOutbound) Example() example.Service { return stubExampleAgg{} }

type stubService struct {
	createItem models.Item
	createErr  *code.Code
	listItems  []models.Item
	listErr    *code.Code
	getItem    models.Item
	getErr     *code.Code
	updateItem models.Item
	updateErr  *code.Code
	deleteErr  *code.Code
}

func (s stubService) CreateItem(context.Context, models.Item) (models.Item, *code.Code) {
	return s.createItem, s.createErr
}
func (s stubService) ListItems(context.Context) ([]models.Item, *code.Code) {
	return s.listItems, s.listErr
}
func (s stubService) GetItem(context.Context, int) (models.Item, *code.Code) {
	return s.getItem, s.getErr
}
func (s stubService) UpdateItem(context.Context, models.Item) (models.Item, *code.Code) {
	return s.updateItem, s.updateErr
}
func (s stubService) DeleteItem(context.Context, int) *code.Code { return s.deleteErr }

// --- helpers ---
func routeCtxWithID(id string) context.Context {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	return context.WithValue(context.Background(), chi.RouteCtxKey, rctx)
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

// Close implements io.Closer.
func (errReader) Close() error { return nil }

// --- tests ---
func TestSendResponse(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	rec := httptest.NewRecorder()
	sendResponse(rec, utils.GenerateSuccessResponse(map[string]string{"a": "b"}))
	if rec.Code != http.StatusOK {
		t.Fatalf("code %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content type %s", ct)
	}
	var resp response.GenericResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	m := resp.Body.(map[string]interface{})
	if m["a"] != "b" {
		t.Fatalf("body %#v", m)
	}

	rec2 := httptest.NewRecorder()
	sendResponse(rec2, &response.HttpResponse{HTTPCode: http.StatusNoContent})
	if rec2.Code != http.StatusNoContent || rec2.Body.Len() != 0 {
		t.Fatalf("no content failed")
	}
}

func TestServiceFromContext(t *testing.T) {
	if svc := serviceFromContext(context.Background()); svc != nil {
		t.Fatalf("expected nil")
	}
	ctx := context.Background()
	ctx = utils.SetRepoCtx(ctx, stubRepo{})
	ctx = utils.SetOutboundCtx(ctx, stubOutbound{})
	ctx = utils.SetLoggerToContext(ctx, stubLogger{})
	if svc := serviceFromContext(ctx); svc == nil {
		t.Fatalf("nil service")
	}
}

func TestListItemsHandler(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	patch := monkey.Patch(serviceFromContext, func(context.Context) services.ServiceImpl {
		return stubService{listItems: []models.Item{{ID: 1, Name: "a"}}}
	})
	defer patch.Unpatch()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	ListItemsHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code %d", rec.Code)
	}
	var resp response.GenericResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	items := resp.Body.([]interface{})
	if len(items) != 1 {
		t.Fatalf("items %#v", items)
	}
}

func TestListItemsHandlerError(t *testing.T) {
	patch := monkey.Patch(serviceFromContext, func(context.Context) services.ServiceImpl {
		return stubService{listErr: &code.ErrInternalServerError}
	})
	defer patch.Unpatch()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	ListItemsHandler(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestListItemsHandlerMissing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	ListItemsHandler(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestCreateItemHandler(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	patch := monkey.Patch(serviceFromContext, func(context.Context) services.ServiceImpl {
		return stubService{createItem: models.Item{ID: 2, Name: "b"}}
	})
	defer patch.Unpatch()
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"name":"b"}`))
	rec := httptest.NewRecorder()
	CreateItemHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestCreateItemHandlerBadBody(t *testing.T) {
	patch := monkey.Patch(serviceFromContext, func(context.Context) services.ServiceImpl { return stubService{} })
	defer patch.Unpatch()
	req := httptest.NewRequest(http.MethodPost, "/", errReader{})
	rec := httptest.NewRecorder()
	CreateItemHandler(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestCreateItemHandlerInvalidJSON(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	patch := monkey.Patch(serviceFromContext, func(context.Context) services.ServiceImpl { return stubService{} })
	defer patch.Unpatch()
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("{"))
	rec := httptest.NewRecorder()
	CreateItemHandler(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestCreateItemHandlerError(t *testing.T) {
	patch := monkey.Patch(serviceFromContext, func(context.Context) services.ServiceImpl {
		return stubService{createErr: &code.ErrInternalServerError}
	})
	defer patch.Unpatch()
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"name":"b"}`))
	rec := httptest.NewRecorder()
	CreateItemHandler(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestCreateItemHandlerMissing(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	CreateItemHandler(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestGetItemHandler(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	patch := monkey.Patch(serviceFromContext, func(context.Context) services.ServiceImpl {
		return stubService{getItem: models.Item{ID: 3, Name: "c"}}
	})
	defer patch.Unpatch()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(routeCtxWithID("3"))
	rec := httptest.NewRecorder()
	GetItemHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestGetItemHandlerBadID(t *testing.T) {
	patch := monkey.Patch(serviceFromContext, func(context.Context) services.ServiceImpl { return stubService{} })
	defer patch.Unpatch()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(routeCtxWithID("bad"))
	rec := httptest.NewRecorder()
	GetItemHandler(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestGetItemHandlerError(t *testing.T) {
	patch := monkey.Patch(serviceFromContext, func(context.Context) services.ServiceImpl {
		return stubService{getErr: &code.ErrItemNotFound}
	})
	defer patch.Unpatch()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(routeCtxWithID("1"))
	rec := httptest.NewRecorder()
	GetItemHandler(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestGetItemHandlerMissing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(routeCtxWithID("1"))
	rec := httptest.NewRecorder()
	GetItemHandler(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestUpdateItemHandler(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	patch := monkey.Patch(serviceFromContext, func(context.Context) services.ServiceImpl {
		return stubService{updateItem: models.Item{ID: 4, Name: "d"}}
	})
	defer patch.Unpatch()
	req := httptest.NewRequest(http.MethodPut, "/", bytes.NewBufferString(`{"name":"d"}`))
	req = req.WithContext(routeCtxWithID("4"))
	rec := httptest.NewRecorder()
	UpdateItemHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestUpdateItemHandlerBadBody(t *testing.T) {
	patch := monkey.Patch(serviceFromContext, func(context.Context) services.ServiceImpl { return stubService{} })
	defer patch.Unpatch()
	req := httptest.NewRequest(http.MethodPut, "/", errReader{})
	req = req.WithContext(routeCtxWithID("1"))
	rec := httptest.NewRecorder()
	UpdateItemHandler(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestUpdateItemHandlerBadID(t *testing.T) {
	patch := monkey.Patch(serviceFromContext, func(context.Context) services.ServiceImpl { return stubService{} })
	defer patch.Unpatch()
	req := httptest.NewRequest(http.MethodPut, "/", bytes.NewBufferString(`{"name":"x"}`))
	req = req.WithContext(routeCtxWithID("bad"))
	rec := httptest.NewRecorder()
	UpdateItemHandler(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestUpdateItemHandlerInvalidJSON(t *testing.T) {
	patch := monkey.Patch(serviceFromContext, func(context.Context) services.ServiceImpl { return stubService{} })
	defer patch.Unpatch()
	req := httptest.NewRequest(http.MethodPut, "/", bytes.NewBufferString("{"))
	req = req.WithContext(routeCtxWithID("1"))
	rec := httptest.NewRecorder()
	UpdateItemHandler(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestUpdateItemHandlerError(t *testing.T) {
	patch := monkey.Patch(serviceFromContext, func(context.Context) services.ServiceImpl {
		return stubService{updateErr: &code.ErrInternalServerError}
	})
	defer patch.Unpatch()
	req := httptest.NewRequest(http.MethodPut, "/", bytes.NewBufferString(`{"name":"x"}`))
	req = req.WithContext(routeCtxWithID("1"))
	rec := httptest.NewRecorder()
	UpdateItemHandler(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestUpdateItemHandlerMissing(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/", nil)
	req = req.WithContext(routeCtxWithID("1"))
	rec := httptest.NewRecorder()
	UpdateItemHandler(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestDeleteItemHandler(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	patch := monkey.Patch(serviceFromContext, func(context.Context) services.ServiceImpl { return stubService{} })
	defer patch.Unpatch()
	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	req = req.WithContext(routeCtxWithID("1"))
	rec := httptest.NewRecorder()
	DeleteItemHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestDeleteItemHandlerBadID(t *testing.T) {
	patch := monkey.Patch(serviceFromContext, func(context.Context) services.ServiceImpl { return stubService{} })
	defer patch.Unpatch()
	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	req = req.WithContext(routeCtxWithID("bad"))
	rec := httptest.NewRecorder()
	DeleteItemHandler(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestDeleteItemHandlerError(t *testing.T) {
	patch := monkey.Patch(serviceFromContext, func(context.Context) services.ServiceImpl {
		return stubService{deleteErr: &code.ErrInternalServerError}
	})
	defer patch.Unpatch()
	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	req = req.WithContext(routeCtxWithID("1"))
	rec := httptest.NewRecorder()
	DeleteItemHandler(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestDeleteItemHandlerMissing(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	req = req.WithContext(routeCtxWithID("1"))
	rec := httptest.NewRecorder()
	DeleteItemHandler(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("code %d", rec.Code)
	}
}
