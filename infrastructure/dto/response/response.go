package response

import "net/http"

// HttpResponse represents the result of an outbound HTTP call.
type HttpResponse struct {
	HTTPCode           int
	ContentType        string
	Headers            map[string]string
	RawResponsePayload interface{}
}

// WithHeader sets a response header.
func (r *HttpResponse) WithHeader(key, value string) *HttpResponse {
	if r.Headers == nil {
		r.Headers = map[string]string{}
	}
	r.Headers[key] = value
	return r
}

// BaseResponse contains status metadata included in every response.
type BaseResponse struct {
	StatusCode    string `json:"statusCode"`
	ReturnMessage string `json:"returnMessage"`
}

// GenericResponse wraps a response body with status metadata.
type GenericResponse[T any] struct {
	BaseResponse
	Body T `json:"body,omitempty"`
}

// ErrorDetail holds a single validation or processing error.
type ErrorDetail struct {
	Location string `json:"location,omitempty"`
	Message  string `json:"message,omitempty"`
	Value    any    `json:"value,omitempty"`
}

// ErrorResponse follows RFC 9457 Problem Details for HTTP APIs.
type ErrorResponse struct {
	Type     string         `json:"type"`
	Status   int            `json:"status"`
	Title    string         `json:"title"`
	Detail   string         `json:"detail"`
	Instance string         `json:"instance,omitempty"`
	Errors   []*ErrorDetail `json:"errors,omitempty"`
}

func NewErrorResponse(status int, detail string) *ErrorResponse {
	return &ErrorResponse{
		Type:   "about:blank",
		Status: status,
		Title:  http.StatusText(status),
		Detail: detail,
	}
}

func (e *ErrorResponse) Error() string   { return e.Detail }
func (e *ErrorResponse) StatusCode() int { return e.Status }
