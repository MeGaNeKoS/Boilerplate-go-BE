package code

import (
	"encoding/json"
	"fmt"
	"net/http"

	"google.golang.org/grpc/codes"

	"project-template/infrastructure/enums"
)

type Code struct {
	HTTPCode  int
	GRPCCode  codes.Code
	KafkaCode string
	Message   string
}

func (r Code) Error() string      { return r.Message }
func (r Code) StatusCode() int    { return r.HTTPCode }

func (r Code) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
	}{Status: r.HTTPCode, Message: r.Message})
}

func (r Code) PrependMessage(prefix string) *Code {
	c := r
	c.Message = fmt.Sprintf("%s: %s", prefix, c.Message)
	return &c
}

func (r Code) AppendMessage(additionalMsg string) *Code {
	c := r
	c.Message = fmt.Sprintf("%s: %s", c.Message, additionalMsg)
	return &c
}

func (r Code) ReplaceMessage(newMessage string) *Code {
	c := r
	c.Message = newMessage
	return &c
}

var (
	// 400 Bad Request

	ErrPayloadError     = &Code{HTTPCode: http.StatusBadRequest, GRPCCode: codes.InvalidArgument, KafkaCode: string(enums.KafkaInvalidArgument), Message: "Payload error"}
	ErrValidationError  = &Code{HTTPCode: http.StatusBadRequest, GRPCCode: codes.InvalidArgument, KafkaCode: string(enums.KafkaInvalidArgument), Message: "Validation error"}
	ErrBadRequest       = &Code{HTTPCode: http.StatusBadRequest, GRPCCode: codes.InvalidArgument, KafkaCode: string(enums.KafkaInvalidArgument), Message: "Bad request"}
	ErrDuplicateRequest = &Code{HTTPCode: http.StatusBadRequest, GRPCCode: codes.InvalidArgument, KafkaCode: string(enums.KafkaInvalidArgument), Message: "Duplicate request"}

	// 401 Unauthorized

	ErrUnauthorized       = &Code{HTTPCode: http.StatusUnauthorized, GRPCCode: codes.Unauthenticated, KafkaCode: string(enums.KafkaUnauthenticated), Message: "Unauthorized"}
	ErrTokenExpired       = &Code{HTTPCode: http.StatusUnauthorized, GRPCCode: codes.Unauthenticated, KafkaCode: string(enums.KafkaUnauthenticated), Message: "Access token expired"}
	ErrInvalidToken       = &Code{HTTPCode: http.StatusUnauthorized, GRPCCode: codes.Unauthenticated, KafkaCode: string(enums.KafkaUnauthenticated), Message: "Invalid token"}
	ErrMissingToken       = &Code{HTTPCode: http.StatusUnauthorized, GRPCCode: codes.Unauthenticated, KafkaCode: string(enums.KafkaUnauthenticated), Message: "Missing token"}
	ErrInvalidCredentials = &Code{HTTPCode: http.StatusUnauthorized, GRPCCode: codes.Unauthenticated, KafkaCode: string(enums.KafkaUnauthenticated), Message: "Invalid credentials"}
	ErrSessionExpired     = &Code{HTTPCode: http.StatusUnauthorized, GRPCCode: codes.Unauthenticated, KafkaCode: string(enums.KafkaUnauthenticated), Message: "Session expired"}

	// 403 Forbidden

	ErrForbidden            = &Code{HTTPCode: http.StatusForbidden, GRPCCode: codes.PermissionDenied, KafkaCode: string(enums.KafkaPermissionDenied), Message: "Forbidden"}
	ErrAccessDenied         = &Code{HTTPCode: http.StatusForbidden, GRPCCode: codes.PermissionDenied, KafkaCode: string(enums.KafkaPermissionDenied), Message: "Access denied"}
	ErrApplicationForbidden = &Code{HTTPCode: http.StatusForbidden, GRPCCode: codes.PermissionDenied, KafkaCode: string(enums.KafkaPermissionDenied), Message: "Application forbidden"}
	ErrUserNotActive        = &Code{HTTPCode: http.StatusForbidden, GRPCCode: codes.PermissionDenied, KafkaCode: string(enums.KafkaPermissionDenied), Message: "User is not active"}

	// 404 Not Found

	ErrNotFound       = &Code{HTTPCode: http.StatusNotFound, GRPCCode: codes.NotFound, KafkaCode: string(enums.KafkaNotFound), Message: "Resource not found"}
	ErrRecordNotFound = &Code{HTTPCode: http.StatusNotFound, GRPCCode: codes.NotFound, KafkaCode: string(enums.KafkaNotFound), Message: "Record not found"}
	ErrItemNotFound   = &Code{HTTPCode: http.StatusNotFound, GRPCCode: codes.NotFound, KafkaCode: string(enums.KafkaNotFound), Message: "Item not found"}

	// 405 Method Not Allowed

	ErrMethodNotAllowed = &Code{HTTPCode: http.StatusMethodNotAllowed, GRPCCode: codes.Unimplemented, KafkaCode: string(enums.KafkaUnimplemented), Message: "Method not allowed"}

	// 409 Conflict

	ErrConflict            = &Code{HTTPCode: http.StatusConflict, GRPCCode: codes.Aborted, KafkaCode: string(enums.KafkaAborted), Message: "Conflict"}
	ErrConstraintViolation = &Code{HTTPCode: http.StatusConflict, GRPCCode: codes.Aborted, KafkaCode: string(enums.KafkaAborted), Message: "Constraint violation"}

	// 413 Payload Too Large

	ErrFileTooLarge = &Code{HTTPCode: http.StatusRequestEntityTooLarge, GRPCCode: codes.ResourceExhausted, KafkaCode: string(enums.KafkaResourceExhausted), Message: "Uploaded file too large"}

	// 415 Unsupported Media Type

	ErrUnsupportedFileType = &Code{HTTPCode: http.StatusUnsupportedMediaType, GRPCCode: codes.InvalidArgument, KafkaCode: string(enums.KafkaInvalidArgument), Message: "Unsupported file type"}

	// 429 Too Many Requests

	ErrTooManyRequests = &Code{HTTPCode: http.StatusTooManyRequests, GRPCCode: codes.ResourceExhausted, KafkaCode: string(enums.KafkaResourceExhausted), Message: "Too many requests"}

	// 500 Internal Server Error

	ErrInternalServerError = &Code{HTTPCode: http.StatusInternalServerError, GRPCCode: codes.Internal, KafkaCode: string(enums.KafkaInternal), Message: "Internal server error"}
	ErrTransactionFailed   = &Code{HTTPCode: http.StatusInternalServerError, GRPCCode: codes.Internal, KafkaCode: string(enums.KafkaInternal), Message: "Transaction failed"}
	ErrDataCorruption      = &Code{HTTPCode: http.StatusInternalServerError, GRPCCode: codes.Internal, KafkaCode: string(enums.KafkaInternal), Message: "Data corruption detected"}
	ErrDeadlockDetected    = &Code{HTTPCode: http.StatusInternalServerError, GRPCCode: codes.Internal, KafkaCode: string(enums.KafkaInternal), Message: "Database deadlock detected"}

	// 502 Bad Gateway

	ErrDependencyFailed = &Code{HTTPCode: http.StatusBadGateway, GRPCCode: codes.Unavailable, KafkaCode: string(enums.KafkaUnavailable), Message: "Upstream dependency failed"}

	// 503 Service Unavailable

	ErrServiceUnavailable  = &Code{HTTPCode: http.StatusServiceUnavailable, GRPCCode: codes.Unavailable, KafkaCode: string(enums.KafkaUnavailable), Message: "Service unavailable"}
	ErrDatabaseUnavailable = &Code{HTTPCode: http.StatusServiceUnavailable, GRPCCode: codes.Unavailable, KafkaCode: string(enums.KafkaUnavailable), Message: "Database unavailable"}

	// 504 Gateway Timeout

	ErrOperationTimedOut = &Code{HTTPCode: http.StatusGatewayTimeout, GRPCCode: codes.DeadlineExceeded, KafkaCode: string(enums.KafkaDeadlineExceeded), Message: "Operation timed out"}

	// Custom Implementation

	ErrCreateRequestUrl          = &Code{HTTPCode: http.StatusInternalServerError, GRPCCode: codes.Internal, KafkaCode: string(enums.KafkaInternal), Message: "Failed to generate url for external service call"}
	ErrCreateRequestPayload      = &Code{HTTPCode: http.StatusInternalServerError, GRPCCode: codes.Internal, KafkaCode: string(enums.KafkaInternal), Message: "Failed to generate payload for external service call"}
	ErrCreateRequestHeaders      = &Code{HTTPCode: http.StatusInternalServerError, GRPCCode: codes.Internal, KafkaCode: string(enums.KafkaInternal), Message: "Failed to generate headers for external service call"}
	ErrFireExternalRequestFailed = &Code{HTTPCode: http.StatusInternalServerError, GRPCCode: codes.Internal, KafkaCode: string(enums.KafkaInternal), Message: "External request failed"}
	ErrReadExternalRequestFailed = &Code{HTTPCode: http.StatusInternalServerError, GRPCCode: codes.Internal, KafkaCode: string(enums.KafkaInternal), Message: "Failed to read external request response"}
)
