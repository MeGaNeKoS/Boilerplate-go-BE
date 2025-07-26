package routes

import (
	"net/http"

	"project-template/pkg/code"
)

// Prefix constants for each API group.
const (
	ItemsPrefix  = "/items"
	SystemPrefix = "/system"
	FilesPrefix  = "/files"
)

// RouteGroup groups related endpoints under a common URL prefix.
type RouteGroup struct {
	Prefix  string
	Routes  *[]RouteDef
	UseAuth bool
}

// RouteGroups lists all API groups in one place for easy iteration.
var RouteGroups = []RouteGroup{
	{Prefix: ItemsPrefix, Routes: &ItemRouteDefs, UseAuth: true},
	{Prefix: SystemPrefix, Routes: &SystemRouteDefs},
	{Prefix: FilesPrefix, Routes: &FileRouteDefs},
}

// ResponseDef describes a single HTTP response.
// ResponseDef describes an HTTP response.
type ResponseDef struct {
	Status      int
	Model       interface{}
	Description string
	ContentType string
	ErrCode     *code.Code
}

// ReqDef describes an HTTP request body and metadata.
type ReqDef struct {
	Model       interface{}
	Description string
	ContentType string
}

// RouteDef describes an HTTP route and related OpenAPI metadata.
type RouteDef struct {
	Method      string
	Pattern     string
	Handler     http.HandlerFunc
	Summary     string
	Description string
	OperationID string
	Tag         string
	Req         *ReqDef
	Responses   []ResponseDef
	Auth        bool
}
