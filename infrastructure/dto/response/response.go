package response

type HttpResponse struct {
	HTTPCode           int
	RawResponsePayload interface{}
}

type BaseResponse struct {
	StatusCode    string `json:"statusCode"`
	ReturnMessage string `json:"returnMessage"`
}

type GenericResponse struct {
	BaseResponse
	Body interface{} `json:"body,omitempty"`
}
