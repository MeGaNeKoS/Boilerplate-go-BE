package response

type HttpResponse struct {
	HTTPCode           int
	ContentType        string
	RawResponsePayload interface{}
}

type BaseResponse struct {
	StatusCode    string `json:"statusCode"`
	ReturnMessage string `json:"returnMessage"`
}

// ProblemDetail represents an error payload following RFC 7807.
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
