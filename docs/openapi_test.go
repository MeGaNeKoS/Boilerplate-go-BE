//go:build docs

package docs

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/swaggest/openapi-go/openapi3"

	"project-template/infrastructure/config"
	"project-template/pkg/code"
	"project-template/server/rest/routes"
)

func TestBuildSpecDefaults(t *testing.T) {
	BuildSpec(nil)
	if Spec.Info.Version != "0.1rc" {
		t.Fatalf("version %s", Spec.Info.Version)
	}
	if len(Spec.Servers) == 0 {
		t.Fatalf("no servers")
	}
	if Spec.Servers[0].Variables["basePath"].Default != "v1" {
		t.Fatalf("base path %s", Spec.Servers[0].Variables["basePath"].Default)
	}
	if len(SpecBytes()) == 0 || len(YAMLSpecBytes()) == 0 {
		t.Fatalf("no bytes")
	}
}

func TestBuildSpecCustom(t *testing.T) {
	cfg := &config.Config{Version: "1.2", REST: config.ListenerConfig{Host: "h", Port: "9"}, Server: config.ServerConfig{Endpoint: config.EndpointConfig{Based: "/api"}}}
	BuildSpec(cfg)
	if Spec.Info.Version != "1.2" {
		t.Fatalf("version %s", Spec.Info.Version)
	}
	host := Spec.Servers[0].Variables["hostname"].Default
	if host != "http://h:9" {
		t.Fatalf("host %s", host)
	}
	if Spec.Servers[0].Variables["basePath"].Default != "api" {
		t.Fatalf("base path %s", Spec.Servers[0].Variables["basePath"].Default)
	}
}

func TestHandler(t *testing.T) {
	BuildSpec(nil)
	h := Handler("/docs")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/docs/openapi.json", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK || len(rr.Body.Bytes()) == 0 {
		t.Fatalf("json failed %d %d", rr.Code, rr.Body.Len())
	}
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/docs/openapi.yaml", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK || len(rr.Body.Bytes()) == 0 {
		t.Fatalf("yaml failed %d %d", rr.Code, rr.Body.Len())
	}
}

func TestAddResponsesContentType(t *testing.T) {
	r := openapi3.NewReflector()
	oc, _ := r.NewOperationContext(http.MethodGet, "/foo")
	addResponses(r, oc, routes.ResponseDef{Status: http.StatusCreated, Model: new(struct{})})
	resp := oc.Response()
	if len(resp) != 1 || resp[0].ContentType != "" || resp[0].HTTPStatus != http.StatusCreated {
		t.Fatalf("unexpected %+v", resp)
	}

	oc2, _ := r.NewOperationContext(http.MethodPost, "/bar")
	addResponses(r, oc2, routes.ResponseDef{Status: http.StatusOK, Model: new(struct{}), ContentType: "text/plain"})
	resp = oc2.Response()
	if len(resp) != 1 || resp[0].ContentType != "text/plain" {
		t.Fatalf("content type %s", resp[0].ContentType)
	}
}

func TestHandlerFallback(t *testing.T) {
	BuildSpec(nil)
	h := Handler("/docs")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/docs/", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status %d", rr.Code)
	}
	if len(rr.Body.Bytes()) == 0 {
		t.Fatalf("no body")
	}
}

func TestAddErrorResponsesCustomize(t *testing.T) {
	saved := config.Cfg
	config.Cfg = &config.Config{AppName: "app"}
	defer func() { config.Cfg = saved }()

	r := openapi3.NewReflector()
	oc, _ := r.NewOperationContext(http.MethodGet, "/x")
	errCode := code.Code{HTTPCode: http.StatusTeapot, InternalCode: 42, Message: "pot"}
	addErrorResponses(r, oc,
		routes.ResponseDef{ErrCode: &errCode, Description: "d"},
		routes.ResponseDef{},
	)

	cus := oc.Response()[0]
	cus.Customize(nil)
	rr := &openapi3.ResponseOrRef{Response: &openapi3.Response{Content: map[string]openapi3.MediaType{"application/json": {}}}}
	cus.Customize(rr)
	if rr.Response.Content["application/json"].Examples == nil {
		t.Fatalf("no examples")
	}
	if _, ok := r.Spec.Components.Examples.MapOfExampleOrRefValues["err_42"]; !ok {
		t.Fatalf("component example missing")
	}
	if err := r.AddOperation(oc); err != nil {
		t.Fatal(err)
	}
}

func TestBuildSpecReqContentType(t *testing.T) {
	oldItems := routes.ItemRouteDefs
	oldSys := routes.SystemRouteDefs
	routes.ItemRouteDefs = []routes.RouteDef{{
		Method:    http.MethodPost,
		Pattern:   "/x",
		Req:       &routes.ReqDef{Model: new(struct{}), ContentType: "text/plain", Description: "d"},
		Responses: []routes.ResponseDef{{Model: new(struct{})}},
	}}
	routes.SystemRouteDefs = nil
	defer func() { routes.ItemRouteDefs = oldItems; routes.SystemRouteDefs = oldSys }()

	BuildSpec(nil)
	if _, ok := Spec.Paths.MapOfPathItemValues["/items/x"]; !ok {
		t.Fatalf("path missing")
	}
}

func TestBuildSpecLeadingSlash(t *testing.T) {
	old := routes.RouteGroups
	routes.RouteGroups = []routes.RouteGroup{{
		Prefix: "",
		Routes: &[]routes.RouteDef{{Method: http.MethodGet, Pattern: "foo", Handler: func(http.ResponseWriter, *http.Request) {}}},
	}}
	defer func() { routes.RouteGroups = old }()
	BuildSpec(nil)
	if _, ok := Spec.Paths.MapOfPathItemValues["/foo"]; !ok {
		t.Fatalf("route not added")
	}
}
