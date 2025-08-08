package transport

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"reflect"
	"time"

	"github.com/google/go-querystring/query"
	"github.com/mohae/deepcopy"

	"project-template/infrastructure/config"
	"project-template/infrastructure/dto/response"
	"project-template/pkg/code"
	"project-template/pkg/logger"
)

type HTTPOutbound struct {
	Path        string            `json:"path"`
	Method      string            `json:"method"`
	Host        string            `json:"host"`
	ServiceName string            `json:"serviceName"`
	Headers     map[string]string `json:"headers,omitempty"`
	QueryParam  interface{}       `json:"queryParam,omitempty"`
	Body        interface{}       `json:"body,omitempty"`
	Response    interface{}       `json:"response,omitempty"`
}

// Copy returns a deep copy of the HTTPOutbound struct, so changes won't affect the original.
func (o *HTTPOutbound) Copy() *HTTPOutbound {
	copied := deepcopy.Copy(o)
	return copied.(*HTTPOutbound)
}

// SendHTTPRequest executes the outbound HTTP request using the provided logger
// for tracing and error reporting. The method returns the decoded response
// payload along with a potential error code if the request fails.
func (o *HTTPOutbound) SendHTTPRequest(log logger.Logger) (response.HttpResponse, *code.Code) {
	url, err := buildURL(*o)
	if err != nil {
		log.ErrorF("Failed to build URL: %v", err)
		return emptyResponse(), code.ErrCreateRequestUrl
	}

	body, err := marshalRequestBody(o.Body)
	if err != nil {
		log.ErrorF("Failed to marshal request body: %v", err)
		return emptyResponse(), code.ErrCreateRequestPayload
	}

	httpReq, err := buildHTTPRequest(*o, url, body)
	if err != nil {
		log.ErrorF("Failed to build HTTP request: %v", err)
		return emptyResponse(), code.ErrCreateRequestHeaders
	}

	httpResp, err := sendHTTPRequest(httpReq)
	if err != nil {
		log.ErrorF("Request failed: %v", err)
		return emptyResponse(), code.ErrFireExternalRequestFailed
	}
	defer func(Body io.ReadCloser) {
		if err := Body.Close(); err != nil {
			log.WarnF("Failed to close response body: %v", err)
		}
	}(httpResp.Body)

	respPlaceholder := reflect.New(reflect.TypeOf(o.Response).Elem()).Interface()
	result, err := handleResponse(httpResp, respPlaceholder, log)
	if err != nil {
		log.ErrorF("Failed to handle response: %v", err)
		return emptyResponse(), code.ErrReadExternalRequestFailed
	}

	return result, nil
}

func buildURL(req HTTPOutbound) (string, error) {
	url := fmt.Sprintf("%s%s", req.Host, req.Path)
	if req.QueryParam != nil {
		q, err := query.Values(req.QueryParam)
		if err != nil {
			return "", err
		}
		url += "?" + q.Encode()
	}
	return url, nil
}

func marshalRequestBody(body interface{}) ([]byte, error) {
	if body == nil {
		return nil, nil
	}
	return json.Marshal(body)
}

func buildHTTPRequest(req HTTPOutbound, url string, body []byte) (*http.Request, error) {
	httpReq, err := http.NewRequest(req.Method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	return httpReq, nil
}

func sendHTTPRequest(req *http.Request) (*http.Response, error) {
	tlsCfg := &tls.Config{InsecureSkipVerify: config.Cfg.Server.SkipTLSVerify}
	client := &http.Client{
		Timeout:   time.Duration(config.Cfg.Server.Timeout.Server) * time.Second,
		Transport: &http.Transport{TLSClientConfig: tlsCfg},
	}
	return client.Do(req)
}

func handleResponse(resp *http.Response, responseContainer interface{}, log logger.Logger) (response.HttpResponse, error) {
	var result response.HttpResponse
	result.HTTPCode = resp.StatusCode
	result.ContentType = resp.Header.Get("Content-Type")

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return response.HttpResponse{}, fmt.Errorf("failed to read response body: %w", err)
	}

	dump, _ := httputil.DumpResponse(resp, true)

	if errMarshal := json.Unmarshal(bodyBytes, responseContainer); errMarshal != nil {
		log.ErrorF("Failed to unmarshal into %T: %v\nDump: %s", responseContainer, errMarshal, dump)
		return response.HttpResponse{}, fmt.Errorf("unmarshal failed: %w", errMarshal)
	}

	result.RawResponsePayload = responseContainer

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.ErrorF("Non-2xx response (%d): %s", resp.StatusCode, dump)
	}

	return result, nil
}

func emptyResponse() response.HttpResponse {
	return response.HttpResponse{}
}
