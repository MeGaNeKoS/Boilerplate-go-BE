package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"

	"project-template/infrastructure/config"
	dtoitem "project-template/infrastructure/dto/item"
	"project-template/infrastructure/dto/response"
	"project-template/infrastructure/utils"
	outbound "project-template/outbound"
	example "project-template/outbound/service/example"
	"project-template/pkg/code"
	"project-template/pkg/logger"
	repoitem "project-template/repositories/item"
	"project-template/server/rest/handlers/resthuma"
)

type mockItemRepo struct {
	create func(context.Context, *dtoitem.Item) error
	list   func(context.Context) ([]dtoitem.Item, error)
	get    func(context.Context, int) (*dtoitem.Item, error)
	update func(context.Context, *dtoitem.Item) error
	delete func(context.Context, int) error
}

func (m mockItemRepo) Create(ctx context.Context, item *dtoitem.Item) error {
	if m.create != nil {
		return m.create(ctx, item)
	}
	return nil
}
func (m mockItemRepo) List(ctx context.Context) ([]dtoitem.Item, error) {
	if m.list != nil {
		return m.list(ctx)
	}
	return nil, nil
}
func (m mockItemRepo) Get(ctx context.Context, id int) (*dtoitem.Item, error) {
	if m.get != nil {
		return m.get(ctx, id)
	}
	return nil, nil
}
func (m mockItemRepo) Update(ctx context.Context, item *dtoitem.Item) error {
	if m.update != nil {
		return m.update(ctx, item)
	}
	return nil
}
func (m mockItemRepo) Delete(ctx context.Context, id int) error {
	if m.delete != nil {
		return m.delete(ctx, id)
	}
	return nil
}

type mockRepoAgg struct{ repo repoitem.Repository }

func (m mockRepoAgg) GetItemRepository() repoitem.Repository { return m.repo }

type noopLogger struct{}

func (noopLogger) DebugF(string, ...interface{}) {}
func (noopLogger) InfoF(string, ...interface{})  {}
func (noopLogger) WarnF(string, ...interface{})  {}
func (noopLogger) ErrorF(string, ...interface{}) {}
func (noopLogger) FatalF(string, ...interface{}) {}
func (noopLogger) ParentID() string              { return "" }
func (noopLogger) ChildID() string               { return "" }

var _ logger.Logger = (*noopLogger)(nil)

type mockExampleOutbound struct {
	fetchByIDCalled bool
}

func (m *mockExampleOutbound) FetchItemByID(ctx context.Context, id int) (dtoitem.Item, error) {
	m.fetchByIDCalled = true
	return dtoitem.Item{}, nil
}
func (m *mockExampleOutbound) FetchItemByFilter(ctx context.Context, filter string) ([]dtoitem.Item, error) {
	return nil, nil
}

type mockExampleService struct{ out example.ExampleOutbound }

func (m mockExampleService) HTTP() example.ExampleOutbound       { return m.out }
func (m mockExampleService) GRPC() example.ExampleGRPCOutbound   { return nil }
func (m mockExampleService) Kafka() example.ExampleKafkaOutbound { return nil }

type mockOutboundAgg struct{ svc example.Service }

func (m mockOutboundAgg) Example() example.Service { return m.svc }

var _ outbound.Impl = (*mockOutboundAgg)(nil)

func setupCtx(repo repoitem.Repository, exOutbound example.ExampleOutbound) context.Context {
	ctx := context.Background()
	ctx = utils.SetRepoCtx(ctx, mockRepoAgg{repo})
	ctx = utils.SetOutboundCtx(ctx, mockOutboundAgg{mockExampleService{exOutbound}})
	ctx = utils.SetLoggerToContext(ctx, noopLogger{})
	return ctx
}

func decodeBody[T any](t *testing.T, resp *resthuma.Response[*response.GenericResponse[T]]) response.GenericResponse[T] {
	rec := httptest.NewRecorder()
	ctx := humatest.NewContext(nil, httptest.NewRequest(http.MethodGet, "/", nil), rec)
	resp.Body(ctx)
	var out response.GenericResponse[T]
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return out
}

func expectInternal(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error")
	}
	se, ok := err.(huma.StatusError)
	if !ok || se.GetStatus() != http.StatusInternalServerError {
		t.Fatalf("unexpected error %#v", err)
	}
}

func TestListItems(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	repo := mockItemRepo{list: func(context.Context) ([]dtoitem.Item, error) {
		return []dtoitem.Item{{ID: 1, Name: "a"}}, nil
	}}
	ctx := setupCtx(repo, &mockExampleOutbound{})
	resp, err := ListItems(ctx, &dtoitem.ListItemsInput{})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	out := decodeBody(t, resp)
	if len(out.Body) != 1 || out.Body[0].Name != "a" {
		t.Fatalf("body %#v", out.Body)
	}
}

func TestListItemsError(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	repo := mockItemRepo{list: func(context.Context) ([]dtoitem.Item, error) {
		return nil, errors.New("fail")
	}}
	ctx := setupCtx(repo, &mockExampleOutbound{})
	if _, err := ListItems(ctx, &dtoitem.ListItemsInput{}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestCreateItem(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	repo := mockItemRepo{create: func(ctx context.Context, item *dtoitem.Item) error {
		item.ID = 5
		return nil
	}}
	ctx := setupCtx(repo, &mockExampleOutbound{})
	in := &dtoitem.CreateItemInput{Body: dtoitem.Item{Name: "x"}}
	resp, err := CreateItem(ctx, in)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if resp.GetHeaders().Get("Location") != "/items/5" {
		t.Fatalf("location %q", resp.GetHeaders().Get("Location"))
	}
	out := decodeBody(t, resp)
	if out.Body.ID != 5 {
		t.Fatalf("body %#v", out.Body)
	}
}

func TestCreateItemError(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	repo := mockItemRepo{create: func(context.Context, *dtoitem.Item) error { return errors.New("boom") }}
	ctx := setupCtx(repo, &mockExampleOutbound{})
	if _, err := CreateItem(ctx, &dtoitem.CreateItemInput{Body: dtoitem.Item{Name: "x"}}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestGetItem(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	repo := mockItemRepo{get: func(ctx context.Context, id int) (*dtoitem.Item, error) {
		return &dtoitem.Item{ID: id, Name: "y"}, nil
	}}
	ctx := setupCtx(repo, &mockExampleOutbound{})
	resp, err := GetItem(ctx, &dtoitem.IDPath{ID: 7})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	out := decodeBody(t, resp)
	if out.Body.ID != 7 || out.Body.Name != "y" {
		t.Fatalf("body %#v", out.Body)
	}
}

func TestGetItemNotFound(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	repo := mockItemRepo{get: func(context.Context, int) (*dtoitem.Item, error) { return nil, code.ErrItemNotFound }}
	ctx := setupCtx(repo, &mockExampleOutbound{})
	if _, err := GetItem(ctx, &dtoitem.IDPath{ID: 1}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestUpdateItem(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	outbound := &mockExampleOutbound{}
	repo := mockItemRepo{update: func(ctx context.Context, item *dtoitem.Item) error { return nil }}
	ctx := setupCtx(repo, outbound)
	in := &dtoitem.UpdateItemInput{ID: 3, Body: dtoitem.Item{Name: "z"}}
	resp, err := UpdateItem(ctx, in)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !outbound.fetchByIDCalled {
		t.Fatalf("expected fetch call")
	}
	out := decodeBody(t, resp)
	if out.Body.ID != 3 || out.Body.Name != "z" {
		t.Fatalf("body %#v", out.Body)
	}
}

func TestUpdateItemError(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	repo := mockItemRepo{update: func(context.Context, *dtoitem.Item) error { return code.ErrItemNotFound }}
	ctx := setupCtx(repo, &mockExampleOutbound{})
	in := &dtoitem.UpdateItemInput{ID: 2, Body: dtoitem.Item{Name: "a"}}
	if _, err := UpdateItem(ctx, in); err == nil {
		t.Fatalf("expected error")
	}
}

func TestDeleteItem(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	repo := mockItemRepo{delete: func(context.Context, int) error { return nil }}
	ctx := setupCtx(repo, &mockExampleOutbound{})
	resp, err := DeleteItem(ctx, &dtoitem.IDPath{ID: 9})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	rec := httptest.NewRecorder()
	ctx2 := humatest.NewContext(nil, httptest.NewRequest(http.MethodDelete, "/", nil), rec)
	resp.Body(ctx2)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestDeleteItemError(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	repo := mockItemRepo{delete: func(context.Context, int) error { return code.ErrItemNotFound }}
	ctx := setupCtx(repo, &mockExampleOutbound{})
	if _, err := DeleteItem(ctx, &dtoitem.IDPath{ID: 1}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestListItemsNoService(t *testing.T) {
	_, err := ListItems(context.Background(), &dtoitem.ListItemsInput{})
	expectInternal(t, err)
}

func TestCreateItemNoService(t *testing.T) {
	_, err := CreateItem(context.Background(), &dtoitem.CreateItemInput{Body: dtoitem.Item{}})
	expectInternal(t, err)
}

func TestGetItemNoService(t *testing.T) {
	_, err := GetItem(context.Background(), &dtoitem.IDPath{ID: 1})
	expectInternal(t, err)
}

func TestUpdateItemNoService(t *testing.T) {
	in := &dtoitem.UpdateItemInput{ID: 1, Body: dtoitem.Item{}}
	_, err := UpdateItem(context.Background(), in)
	expectInternal(t, err)
}

func TestDeleteItemNoService(t *testing.T) {
	_, err := DeleteItem(context.Background(), &dtoitem.IDPath{ID: 1})
	expectInternal(t, err)
}

func TestServiceFromContextMissing(t *testing.T) {
	if svc := serviceFromContext(context.Background()); svc != nil {
		t.Fatalf("expected nil service")
	}
}
