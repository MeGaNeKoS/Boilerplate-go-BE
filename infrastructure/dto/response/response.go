package response

type HttpResponse struct {
	HTTPCode           int
	ContentType        string
	Headers            map[string]string
	RawResponsePayload interface{}
}

type BaseResponse struct {
	StatusCode    string `json:"statusCode"`
	ReturnMessage string `json:"returnMessage"`
}

// WithHeader sets a response header.
func (r *HttpResponse) WithHeader(key, value string) *HttpResponse {
	if r.Headers == nil {
		r.Headers = map[string]string{}
	}
	r.Headers[key] = value
	return r
}

// WithInstance sets the ProblemDetail instance field when the payload is a
// ProblemDetail. It is a no-op for other payload types.
func (r *HttpResponse) WithInstance(instance string) *HttpResponse {
	if pd, ok := r.RawResponsePayload.(ProblemDetail); ok {
		pd.Instance = instance
		r.RawResponsePayload = pd
	}
	return r
}

// ProblemDetail represents an error payload following RFC 9457.
type ProblemDetail struct {
	Type     string `json:"type,omitempty"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`
	Code     string `json:"code,omitempty"`
}

// GenericResponse wraps a response body with status metadata.
// The fields are flattened so status information lives alongside the body.
type GenericResponse[T any] struct {
	BaseResponse
	Body T `json:"body,omitempty"`
}
