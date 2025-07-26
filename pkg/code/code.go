package code

import (
	"fmt"
	"net/http"

	"google.golang.org/grpc/codes"

	"project-template/infrastructure/enums"
)

type Code struct {
	InternalCode int
	Message      string
	HTTPCode     int
	GRPCCode     codes.Code
	KafkaCode    string
}

// Error implements the error interface, returning the message.
func (r Code) Error() string {
	return r.Message
}

func (r Code) PrependMessage(prefix string) *Code {
	c := r
	c.Message = fmt.Sprintf("%s: %s", prefix, c.Message)
	return &c
}

func (r Code) AppendMessage(additionalMsg string) *Code {
	r.Message = fmt.Sprintf("%s: %s", r.Message, additionalMsg)
	return &r
}

func (r Code) ReplaceMessage(newMessage string) *Code {
	r.Message = newMessage
	return &r
}

var (
	// ErrPayloadError 400 Bad Request
	ErrPayloadError     = Code{HTTPCode: http.StatusBadRequest, GRPCCode: codes.InvalidArgument, KafkaCode: string(enums.KafkaInvalidArgument), Message: "Payload error", InternalCode: 1}
	ErrValidationError  = Code{HTTPCode: http.StatusBadRequest, GRPCCode: codes.InvalidArgument, KafkaCode: string(enums.KafkaInvalidArgument), Message: "Validation error", InternalCode: 2}
	ErrBadRequest       = Code{HTTPCode: http.StatusBadRequest, GRPCCode: codes.InvalidArgument, KafkaCode: string(enums.KafkaInvalidArgument), Message: "Bad request", InternalCode: 3}
	ErrDuplicateRequest = Code{HTTPCode: http.StatusBadRequest, GRPCCode: codes.InvalidArgument, KafkaCode: string(enums.KafkaInvalidArgument), Message: "Duplicate request", InternalCode: 4}

	// 401 Unauthorized
	ErrUnauthorized       = Code{HTTPCode: http.StatusUnauthorized, GRPCCode: codes.Unauthenticated, KafkaCode: string(enums.KafkaUnauthenticated), Message: "Unauthorized", InternalCode: 5}
	ErrTokenExpired       = Code{HTTPCode: http.StatusUnauthorized, GRPCCode: codes.Unauthenticated, KafkaCode: string(enums.KafkaUnauthenticated), Message: "Access token expired", InternalCode: 6}
	ErrInvalidToken       = Code{HTTPCode: http.StatusUnauthorized, GRPCCode: codes.Unauthenticated, KafkaCode: string(enums.KafkaUnauthenticated), Message: "Invalid token", InternalCode: 7}
	ErrMissingToken       = Code{HTTPCode: http.StatusUnauthorized, GRPCCode: codes.Unauthenticated, KafkaCode: string(enums.KafkaUnauthenticated), Message: "Missing token", InternalCode: 8}
	ErrInvalidCredentials = Code{HTTPCode: http.StatusUnauthorized, GRPCCode: codes.Unauthenticated, KafkaCode: string(enums.KafkaUnauthenticated), Message: "Invalid credentials", InternalCode: 9}
	ErrSessionExpired     = Code{HTTPCode: http.StatusUnauthorized, GRPCCode: codes.Unauthenticated, KafkaCode: string(enums.KafkaUnauthenticated), Message: "Session expired", InternalCode: 10}

	// 403 Forbidden
	ErrForbidden            = Code{HTTPCode: http.StatusForbidden, GRPCCode: codes.PermissionDenied, KafkaCode: string(enums.KafkaPermissionDenied), Message: "Forbidden", InternalCode: 11}
	ErrAccessDenied         = Code{HTTPCode: http.StatusForbidden, GRPCCode: codes.PermissionDenied, KafkaCode: string(enums.KafkaPermissionDenied), Message: "Access denied", InternalCode: 12}
	ErrApplicationForbidden = Code{HTTPCode: http.StatusForbidden, GRPCCode: codes.PermissionDenied, KafkaCode: string(enums.KafkaPermissionDenied), Message: "Application forbidden", InternalCode: 13}
	ErrUserNotActive        = Code{HTTPCode: http.StatusForbidden, GRPCCode: codes.PermissionDenied, KafkaCode: string(enums.KafkaPermissionDenied), Message: "User is not active", InternalCode: 14}

	// 404 Not Found
	ErrNotFound       = Code{HTTPCode: http.StatusNotFound, GRPCCode: codes.NotFound, KafkaCode: string(enums.KafkaNotFound), Message: "Resource not found", InternalCode: 15}
	ErrRecordNotFound = Code{HTTPCode: http.StatusNotFound, GRPCCode: codes.NotFound, KafkaCode: string(enums.KafkaNotFound), Message: "Record not found", InternalCode: 16}

	// Item not found in repository
	ErrItemNotFound = Code{HTTPCode: http.StatusNotFound, GRPCCode: codes.NotFound, KafkaCode: string(enums.KafkaNotFound), Message: "Item not found", InternalCode: 17}

	// 405 Method Not Allowed
	ErrMethodNotAllowed = Code{HTTPCode: http.StatusMethodNotAllowed, GRPCCode: codes.Unimplemented, KafkaCode: string(enums.KafkaUnimplemented), Message: "Method not allowed", InternalCode: 18}

	// 409 Conflict
	ErrConflict            = Code{HTTPCode: http.StatusConflict, GRPCCode: codes.Aborted, KafkaCode: string(enums.KafkaAborted), Message: "Conflict", InternalCode: 19}
	ErrConstraintViolation = Code{HTTPCode: http.StatusConflict, GRPCCode: codes.Aborted, KafkaCode: string(enums.KafkaAborted), Message: "Constraint violation", InternalCode: 20}

	// 413 Payload Too Large
	ErrFileTooLarge = Code{HTTPCode: http.StatusRequestEntityTooLarge, GRPCCode: codes.ResourceExhausted, KafkaCode: string(enums.KafkaResourceExhausted), Message: "Uploaded file too large", InternalCode: 21}

	// 415 Unsupported Media Type
	ErrUnsupportedFileType = Code{HTTPCode: http.StatusUnsupportedMediaType, GRPCCode: codes.InvalidArgument, KafkaCode: string(enums.KafkaInvalidArgument), Message: "Unsupported file type", InternalCode: 22}

	// 429 Too Many Requests
	ErrTooManyRequests = Code{HTTPCode: http.StatusTooManyRequests, GRPCCode: codes.ResourceExhausted, KafkaCode: string(enums.KafkaResourceExhausted), Message: "Too many requests", InternalCode: 23}

	// 500 Internal Server Error
	ErrInternalServerError = Code{HTTPCode: http.StatusInternalServerError, GRPCCode: codes.Internal, KafkaCode: string(enums.KafkaInternal), Message: "Internal server error", InternalCode: 24}
	ErrTransactionFailed   = Code{HTTPCode: http.StatusInternalServerError, GRPCCode: codes.Internal, KafkaCode: string(enums.KafkaInternal), Message: "Transaction failed", InternalCode: 25}
	ErrDataCorruption      = Code{HTTPCode: http.StatusInternalServerError, GRPCCode: codes.Internal, KafkaCode: string(enums.KafkaInternal), Message: "Data corruption detected", InternalCode: 26}
	ErrDeadlockDetected    = Code{HTTPCode: http.StatusInternalServerError, GRPCCode: codes.Internal, KafkaCode: string(enums.KafkaInternal), Message: "Database deadlock detected", InternalCode: 27}

	// 502 Bad Gateway
	ErrDependencyFailed = Code{HTTPCode: http.StatusBadGateway, GRPCCode: codes.Unavailable, KafkaCode: string(enums.KafkaUnavailable), Message: "Upstream dependency failed", InternalCode: 28}

	// 503 Service Unavailable
	ErrServiceUnavailable  = Code{HTTPCode: http.StatusServiceUnavailable, GRPCCode: codes.Unavailable, KafkaCode: string(enums.KafkaUnavailable), Message: "Service unavailable", InternalCode: 29}
	ErrDatabaseUnavailable = Code{HTTPCode: http.StatusServiceUnavailable, GRPCCode: codes.Unavailable, KafkaCode: string(enums.KafkaUnavailable), Message: "Database unavailable", InternalCode: 30}

	// 504 Gateway Timeout
	ErrOperationTimedOut = Code{HTTPCode: http.StatusGatewayTimeout, GRPCCode: codes.DeadlineExceeded, KafkaCode: string(enums.KafkaDeadlineExceeded), Message: "Operation timed out", InternalCode: 31}

	// Custom Implementation
	ErrCreateRequestUrl          = Code{HTTPCode: http.StatusInternalServerError, GRPCCode: codes.Internal, KafkaCode: string(enums.KafkaInternal), Message: "Failed to generate url for external service call", InternalCode: 32}
	ErrCreateRequestPayload      = Code{HTTPCode: http.StatusInternalServerError, GRPCCode: codes.Internal, KafkaCode: string(enums.KafkaInternal), Message: "Failed to generate payload for external service call", InternalCode: 33}
	ErrCreateRequestHeaders      = Code{HTTPCode: http.StatusInternalServerError, GRPCCode: codes.Internal, KafkaCode: string(enums.KafkaInternal), Message: "Failed to generate headers for external service call", InternalCode: 34}
	ErrFireExternalRequestFailed = Code{HTTPCode: http.StatusInternalServerError, GRPCCode: codes.Internal, KafkaCode: string(enums.KafkaInternal), Message: "External request failed", InternalCode: 35}
	ErrReadExternalRequestFailed = Code{HTTPCode: http.StatusInternalServerError, GRPCCode: codes.Internal, KafkaCode: string(enums.KafkaInternal), Message: "Failed to read external request response", InternalCode: 36}
)
