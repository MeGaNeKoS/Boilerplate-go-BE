package item

import (
	"context"
	"errors"
	models "project-template/infrastructure/dto/item"
	"project-template/outbound/service/example"
	"project-template/pkg/code"
	"project-template/pkg/logger"
	repoitem "project-template/repositories/item"
	"testing"
)

type stubLogger struct{}

func (stubLogger) DebugF(string, ...interface{}) {}
func (stubLogger) InfoF(string, ...interface{})  {}
func (stubLogger) WarnF(string, ...interface{})  {}
func (stubLogger) ErrorF(string, ...interface{}) {}
func (stubLogger) FatalF(string, ...interface{}) {}
func (stubLogger) ParentID() string              { return "" }
func (stubLogger) ChildID() string               { return "" }
func (stubLogger) CloseLogFile()                 {}

// --- repository stubs ---

type stubItemRepo struct {
	createCalled bool
	createErr    error
	listResult   []models.Item
	listErr      error
	getResult    *models.Item
	getErr       error
	updateErr    error
	deleteErr    error
}

func (s *stubItemRepo) Create(_ context.Context, item *models.Item) error {
	s.createCalled = true
	if s.createErr != nil {
		return s.createErr
	}
	item.ID = 1
	return nil
}
func (s *stubItemRepo) List(_ context.Context) ([]models.Item, error) {
	return s.listResult, s.listErr
}
func (s *stubItemRepo) Get(_ context.Context, id int) (*models.Item, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.getResult, nil
}
func (s *stubItemRepo) Update(_ context.Context, item *models.Item) error {
	return s.updateErr
}
func (s *stubItemRepo) Delete(_ context.Context, id int) error {
	return s.deleteErr
}

// stubTxn implements the Transaction interface for testing.

// --- outbound stubs ---

type stubExternal struct {
	called bool
	err    error
}

func (s *stubExternal) FetchItemByID(_ context.Context, id int) (models.Item, error) {
	s.called = true
	if s.err != nil {
		return models.Item{}, s.err
	}
	return models.Item{ID: id}, nil
}

func (s *stubExternal) FetchItemByFilter(_ context.Context, f string) ([]models.Item, error) {
	s.called = true
	if s.err != nil {
		return nil, s.err
	}
	return []models.Item{{ID: 1}}, nil
}

func TestCreateItem(t *testing.T) {
	repo := &stubItemRepo{}
	svc := NewService(repo, &stubExternal{}, stubLogger{})
	item, errCode := svc.CreateItem(context.Background(), models.Item{Name: "a"})
	if errCode != nil || item.ID != 1 || !repo.createCalled {
		t.Fatalf("unexpected result %#v code %#v", item, errCode)
	}
}

func TestCreateItemError(t *testing.T) {
	repo := &stubItemRepo{createErr: errors.New("boom")}
	svc := NewService(repo, &stubExternal{}, stubLogger{})
	item, errCode := svc.CreateItem(context.Background(), models.Item{Name: "b"})
	if errCode != &code.ErrInternalServerError || item.ID != 0 {
		t.Fatalf("expected error, got item %#v code %#v", item, errCode)
	}
}

func TestListItemsError(t *testing.T) {
	repo := &stubItemRepo{listErr: errors.New("oops")}
	svc := NewService(repo, &stubExternal{}, stubLogger{})
	_, errCode := svc.ListItems(context.Background())
	if errCode != &code.ErrInternalServerError {
		t.Fatalf("expected internal error, got %#v", errCode)
	}
}

func TestListItems(t *testing.T) {
	repo := &stubItemRepo{listResult: []models.Item{{ID: 1, Name: "a"}}}
	svc := NewService(repo, &stubExternal{}, stubLogger{})
	list, errCode := svc.ListItems(context.Background())
	if errCode != nil || len(list) != 1 || list[0].ID != 1 {
		t.Fatalf("unexpected list %#v code %#v", list, errCode)
	}
}

func TestGetItemNotFound(t *testing.T) {
	repo := &stubItemRepo{getErr: &code.ErrItemNotFound}
	svc := NewService(repo, &stubExternal{}, stubLogger{})
	_, errCode := svc.GetItem(context.Background(), 5)
	if errCode != &code.ErrItemNotFound {
		t.Fatalf("expected not found, got %#v", errCode)
	}
}

func TestGetItem(t *testing.T) {
	repo := &stubItemRepo{getResult: &models.Item{ID: 3, Name: "c"}}
	svc := NewService(repo, &stubExternal{}, stubLogger{})
	item, errCode := svc.GetItem(context.Background(), 3)
	if errCode != nil || item.ID != 3 {
		t.Fatalf("unexpected result %#v code %#v", item, errCode)
	}
}

func TestGetItemError(t *testing.T) {
	repo := &stubItemRepo{getErr: errors.New("db down")}
	svc := NewService(repo, &stubExternal{}, stubLogger{})
	_, errCode := svc.GetItem(context.Background(), 3)
	if errCode != &code.ErrInternalServerError {
		t.Fatalf("expected internal error, got %#v", errCode)
	}
}

func TestUpdateItemCallsOutbound(t *testing.T) {
	repo := &stubItemRepo{}
	out := &stubExternal{}
	svc := NewService(repo, out, stubLogger{})
	_, errCode := svc.UpdateItem(context.Background(), models.Item{ID: 2})
	if errCode != nil || !out.called {
		t.Fatalf("unexpected result code %#v outbound called %v", errCode, out.called)
	}
}

func TestUpdateItemNotFound(t *testing.T) {
	repo := &stubItemRepo{updateErr: &code.ErrItemNotFound}
	out := &stubExternal{}
	svc := NewService(repo, out, stubLogger{})
	_, errCode := svc.UpdateItem(context.Background(), models.Item{ID: 2})
	if errCode != &code.ErrItemNotFound {
		t.Fatalf("expected not found, got %#v", errCode)
	}
}

func TestUpdateItemError(t *testing.T) {
	repo := &stubItemRepo{updateErr: errors.New("fail")}
	out := &stubExternal{}
	svc := NewService(repo, out, stubLogger{})
	_, errCode := svc.UpdateItem(context.Background(), models.Item{ID: 2})
	if errCode != &code.ErrInternalServerError {
		t.Fatalf("expected internal error, got %#v", errCode)
	}
}

func TestUpdateItemOutboundError(t *testing.T) {
	repo := &stubItemRepo{}
	out := &stubExternal{err: errors.New("bad")}
	svc := NewService(repo, out, stubLogger{})
	_, errCode := svc.UpdateItem(context.Background(), models.Item{ID: 10})
	if errCode != nil || !out.called {
		t.Fatalf("unexpected code %#v outbound called %v", errCode, out.called)
	}
}

func TestDeleteItem(t *testing.T) {
	repo := &stubItemRepo{}
	svc := NewService(repo, &stubExternal{}, stubLogger{})
	errCode := svc.DeleteItem(context.Background(), 1)
	if errCode != nil {
		t.Fatalf("unexpected code %#v", errCode)
	}
}

func TestDeleteItemNotFound(t *testing.T) {
	repo := &stubItemRepo{deleteErr: &code.ErrItemNotFound}
	svc := NewService(repo, &stubExternal{}, stubLogger{})
	errCode := svc.DeleteItem(context.Background(), 1)
	if errCode != &code.ErrItemNotFound {
		t.Fatalf("expected not found, got %#v", errCode)
	}
}

func TestDeleteItemError(t *testing.T) {
	repo := &stubItemRepo{deleteErr: errors.New("fail")}
	svc := NewService(repo, &stubExternal{}, stubLogger{})
	errCode := svc.DeleteItem(context.Background(), 1)
	if errCode != &code.ErrInternalServerError {
		t.Fatalf("expected internal error, got %#v", errCode)
	}
}

func TestNewServiceMissingDependencies(t *testing.T) {
	cases := []struct {
		name string
		repo repoitem.Repository
		ext  example.ExampleOutbound
		log  logger.Logger
	}{
		{"nil repo", nil, &stubExternal{}, stubLogger{}},
		{"nil external", &stubItemRepo{}, nil, stubLogger{}},
		{"nil logger", &stubItemRepo{}, &stubExternal{}, nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil || r != "missing dependencies" {
					t.Fatalf("expected panic")
				}
			}()
			_ = NewService(tc.repo, tc.ext, tc.log)
		})
	}
}
