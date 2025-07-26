//go:build grpc

package grpc_test

import (
	"context"
	"database/sql"
	"net"
	"net/http"
	"reflect"
	"testing"
	"time"
	"unsafe"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/bouk/monkey"
	"project-template/cmd"
	"project-template/infrastructure/config"
	"project-template/infrastructure/dto/response"
	"project-template/infrastructure/utils"
	"project-template/outbound/transport"
	pb "project-template/pb"
	codepkg "project-template/pkg/code"
	"project-template/pkg/logger"
	"project-template/test/testutil"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func setupServer(t *testing.T) (pb.ItemServiceClient, sqlmock.Sqlmock, string) {
	t.Helper()
	dir := t.TempDir()
	priv, pub := testutil.WriteKeyFiles(t, dir)
	config.Cfg = &config.Config{
		AppName:   "APP",
		Server:    config.ServerConfig{JWT: config.JWTConfig{PrivateKey: priv, PublicKey: pub}},
		GRPC:      config.ListenerConfig{Host: "127.0.0.1", Port: "0"},
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
		return response.HttpResponse{HTTPCode: http.StatusOK, RawResponsePayload: &response.GenericResponse{}}, nil
	})
	t.Cleanup(patchHTTP.Unpatch)

	srv := cmd.GetGRPCServer(config.Cfg, testutil.StubLogger{})
	go func() { _ = srv.StartServer() }()
	var ln net.Listener
	for i := 0; i < 50; i++ {
		sv := reflect.ValueOf(srv).Elem().FieldByName("listener")
		tmp := reflect.NewAt(sv.Type(), unsafe.Pointer(sv.UnsafeAddr())).Elem().Interface()
		if l, ok := tmp.(net.Listener); ok && l != nil {
			ln = l
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if ln == nil {
		t.Fatal("listener not ready")
	}

	conn, err := grpc.Dial(ln.Addr().String(), grpc.WithInsecure())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return pb.NewItemServiceClient(conn), mock, tok
}

func authCtx(tok string) context.Context {
	return metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+tok))
}

func TestCreateItemIntegrationGRPC(t *testing.T) {
	client, mock, tok := setupServer(t)

	mock.ExpectBegin()
	mock.ExpectExec("INSERT").WillReturnResult(sqlmock.NewResult(3, 1))
	mock.ExpectCommit()

	resp, err := client.CreateItem(authCtx(tok), &pb.Item{Name: "foo"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetId() != 3 || resp.GetName() != "foo" {
		t.Fatalf("unexpected %#v", resp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestListItemsIntegrationGRPC(t *testing.T) {
	client, mock, tok := setupServer(t)
	rows := sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "foo")
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	resp, err := client.ListItems(authCtx(tok), &pb.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.GetItems()) != 1 || resp.GetItems()[0].GetId() != 1 {
		t.Fatalf("unexpected %#v", resp.GetItems())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGetItemIntegrationGRPC(t *testing.T) {
	client, mock, tok := setupServer(t)
	rows := sqlmock.NewRows([]string{"id", "name"}).AddRow(2, "bar")
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	resp, err := client.GetItem(authCtx(tok), &pb.ItemID{Id: 2})
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetId() != 2 || resp.GetName() != "bar" {
		t.Fatalf("unexpected %#v", resp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateItemIntegrationGRPC(t *testing.T) {
	client, mock, tok := setupServer(t)
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	resp, err := client.UpdateItem(authCtx(tok), &pb.Item{Id: 4, Name: "baz"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetId() != 4 || resp.GetName() != "baz" {
		t.Fatalf("unexpected %#v", resp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteItemIntegrationGRPC(t *testing.T) {
	client, mock, tok := setupServer(t)
	mock.ExpectBegin()
	mock.ExpectExec("DELETE").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	_, err := client.DeleteItem(authCtx(tok), &pb.ItemID{Id: 5})
	if err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCreateItemIntegrationGRPCDBError(t *testing.T) {
	client, mock, tok := setupServer(t)
	mock.ExpectBegin()
	mock.ExpectExec("INSERT").WillReturnError(sql.ErrConnDone)
	mock.ExpectRollback()

	_, err := client.CreateItem(authCtx(tok), &pb.Item{Name: "x"})
	if status.Code(err) != codes.Internal {
		t.Fatalf("code %v", status.Code(err))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGetItemIntegrationGRPCNotFound(t *testing.T) {
	client, mock, tok := setupServer(t)
	mock.ExpectQuery("SELECT").WillReturnError(sql.ErrNoRows)

	_, err := client.GetItem(authCtx(tok), &pb.ItemID{Id: 9})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("code %v", status.Code(err))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestListItemsIntegrationGRPCUnauthorized(t *testing.T) {
	client, _, _ := setupServer(t)
	_, err := client.ListItems(context.Background(), &pb.Empty{})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("code %v", status.Code(err))
	}
}
