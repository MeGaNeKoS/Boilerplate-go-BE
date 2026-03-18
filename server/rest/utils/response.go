package utils

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

// Response wraps a response body with a status code and arbitrary headers.
type Response[T any] struct {
	Status int
	Body   func(huma.Context)

	headers http.Header
}

// NewResponse creates a new Response with the given HTTP status and body.
func NewResponse[T any](status int, body T) *Response[T] {
	r := &Response[T]{Status: status, headers: http.Header{}}
	r.Body = func(ctx huma.Context) {
		for k, values := range r.headers {
			for i, v := range values {
				if i == 0 {
					ctx.SetHeader(k, v)
				} else {
					ctx.AppendHeader(k, v)
				}
			}
		}
		if r.headers.Get("Content-Type") == "" {
			ctx.SetHeader("Content-Type", "application/json")
		}
		ctx.SetStatus(r.Status)
		switch b := any(body).(type) {
		case []byte:
			_, _ = ctx.BodyWriter().Write(b)
		case string:
			_, _ = ctx.BodyWriter().Write([]byte(b))
		case io.Reader:
			_, _ = io.Copy(ctx.BodyWriter(), b)
			if c, ok := b.(io.Closer); ok {
				_ = c.Close()
			}
		default:
			_ = json.NewEncoder(ctx.BodyWriter()).Encode(b)
		}
	}
	return r
}

// Header sets a header on the response.
func (r *Response[T]) Header(key, value string) *Response[T] {
	r.headers.Add(key, value)
	return r
}

// GetHeaders returns the response headers.
func (r *Response[T]) GetHeaders() http.Header { return r.headers }
