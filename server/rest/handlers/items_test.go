package handlers

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"project-template/infrastructure/config"
	dtoitem "project-template/infrastructure/dto/item"
	"project-template/infrastructure/utils"
	"project-template/outbound"
	"project-template/outbound/service/example"
	"project-template/pkg/code"
	"project-template/pkg/logger"
	repoitem "project-template/repositories/item"
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

func (noopLogger) Debug(string)                  {}
func (noopLogger) DebugF(string, ...interface{}) {}
func (noopLogger) Info(string)                    {}
func (noopLogger) InfoF(string, ...interface{})  {}
func (noopLogger) Warn(string)                    {}
func (noopLogger) WarnF(string, ...interface{})  {}
func (noopLogger) Error(string)                   {}
func (noopLogger) ErrorF(string, ...interface{}) {}
func (noopLogger) Fatal(string)                   {}
func (noopLogger) FatalF(string, ...interface{}) {}
func (noopLogger) ParentID() string              { return "" }
func (noopLogger) ChildID() string               { return "" }

var _ logger.Logger = (*noopLogger)(nil)

type mockExampleOutbound struct {
	fetchByIDCalled bool
}

func (m *mockExampleOutbound) FetchItemByID(_ context.Context, _ int) (dtoitem.Item, error) {
	m.fetchByIDCalled = true
	return dtoitem.Item{}, nil
}
func (m *mockExampleOutbound) FetchItemByFilter(_ context.Context, _ string) ([]dtoitem.Item, error) {
	return nil, nil
}

type mockExampleService struct{ out example.Outbound }

func (m mockExampleService) HTTP() example.Outbound       { return m.out }
func (m mockExampleService) GRPC() example.GrpcOutbound   { return nil }
func (m mockExampleService) Kafka() example.KafkaOutbound { return nil }

type mockOutboundAgg struct{ svc example.Service }

func (m mockOutboundAgg) Example() example.Service { return m.svc }

var _ outbound.Impl = (*mockOutboundAgg)(nil)

func setupCtx(repo repoitem.Repository, exOutbound example.Outbound) context.Context {
	ctx := context.Background()
	ctx = utils.SetRepoCtx(ctx, mockRepoAgg{repo})
	ctx = utils.SetOutboundCtx(ctx, mockOutboundAgg{mockExampleService{exOutbound}})
	ctx = utils.SetLoggerToContext(ctx, noopLogger{})
	return ctx
}

func expectInternal(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error")
	}
	var c *code.Code
	if !errors.As(err, &c) || c.HTTPCode != http.StatusInternalServerError {
		t.Fatalf("unexpected error %#v", err)
	}
}

func TestListItems(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	repo := mockItemRepo{list: func(context.Context) ([]dtoitem.Item, error) {
		return []dtoitem.Item{{ID: 1, Name: "sample"}}, nil
	}}
	ctx := setupCtx(repo, &mockExampleOutbound{})
	resp, err := ListItems(ctx, &dtoitem.ListItemsInput{})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	out := resp.Body
	if len(out.Body) != 1 || out.Body[0].Name != "sample" {
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
	in := &dtoitem.CreateItemInput{Body: dtoitem.Item{Name: "sample"}}
	resp, err := CreateItem(ctx, in)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if resp.Location != "/items/5" {
		t.Fatalf("location %q", resp.Location)
	}
	out := resp.Body
	if out.Body.ID != 5 {
		t.Fatalf("body %#v", out.Body)
	}
}

func TestCreateItemError(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	repo := mockItemRepo{create: func(context.Context, *dtoitem.Item) error { return errors.New("boom") }}
	ctx := setupCtx(repo, &mockExampleOutbound{})
	if _, err := CreateItem(ctx, &dtoitem.CreateItemInput{Body: dtoitem.Item{Name: "sample"}}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestGetItem(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	repo := mockItemRepo{get: func(ctx context.Context, id int) (*dtoitem.Item, error) {
		return &dtoitem.Item{ID: id, Name: "example"}, nil
	}}
	ctx := setupCtx(repo, &mockExampleOutbound{})
	resp, err := GetItem(ctx, &dtoitem.IDPath{ID: 7})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	out := resp.Body
	if out.Body.ID != 7 || out.Body.Name != "example" {
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
	o := &mockExampleOutbound{}
	repo := mockItemRepo{update: func(ctx context.Context, item *dtoitem.Item) error { return nil }}
	ctx := setupCtx(repo, o)
	in := &dtoitem.UpdateItemInput{ID: 3, Body: dtoitem.Item{Name: "example"}}
	resp, err := UpdateItem(ctx, in)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !o.fetchByIDCalled {
		t.Fatalf("expected fetch call")
	}
	out := resp.Body
	if out.Body.ID != 3 || out.Body.Name != "example" {
		t.Fatalf("body %#v", out.Body)
	}
}

func TestUpdateItemError(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	repo := mockItemRepo{update: func(context.Context, *dtoitem.Item) error { return code.ErrItemNotFound }}
	ctx := setupCtx(repo, &mockExampleOutbound{})
	in := &dtoitem.UpdateItemInput{ID: 2, Body: dtoitem.Item{Name: "sample"}}
	if _, err := UpdateItem(ctx, in); err == nil {
		t.Fatalf("expected error")
	}
}

func TestDeleteItem(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	repo := mockItemRepo{delete: func(context.Context, int) error { return nil }}
	ctx := setupCtx(repo, &mockExampleOutbound{})
	_, err := DeleteItem(ctx, &dtoitem.IDPath{ID: 9})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
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
