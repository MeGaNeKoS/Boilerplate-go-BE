package example

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"project-template/infrastructure/config"
	models "project-template/infrastructure/dto/item"
	"project-template/infrastructure/dto/response"
	"project-template/infrastructure/utils"
	"project-template/outbound/transport"
	"project-template/pkg/logger"
)

type exampleOutbound struct {
	log logger.Logger
}

// NewExampleOutbound creates a new outbound client for the Example service.
func NewExampleOutbound(logger logger.Logger) ExampleOutbound {
	return &exampleOutbound{
		log: logger,
	}
}

func (o *exampleOutbound) headers(ctx context.Context) map[string]string {
	tok, _ := utils.GetTokenCtx(ctx)
	return map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", string(tok)),
		"Parent-Id":     o.log.ParentID(),
	}
}

// FetchItemByID performs a GET request to retrieve an item.
func (o *exampleOutbound) FetchItemByID(ctx context.Context, id int) (models.Item, error) {
	req := transport.HTTPOutbound{
		Host:     config.Cfg.Service.Example.Host,
		Path:     fmt.Sprintf("/items/%d", id),
		Method:   http.MethodGet,
		Headers:  o.headers(ctx),
		Response: &response.GenericResponse{},
	}

	resp, code := req.SendHTTPRequest(o.log)
	if code != nil {
		return models.Item{}, fmt.Errorf("external request failed: %s", code.Message)
	}

	generic, ok := resp.RawResponsePayload.(*response.GenericResponse)
	if !ok {
		return models.Item{}, fmt.Errorf("unexpected response type")
	}

	data, err := json.Marshal(generic.Body)
	if err != nil {
		return models.Item{}, err
	}

	var item models.Item
	if err := json.Unmarshal(data, &item); err != nil {
		return models.Item{}, err
	}
	return item, nil
}

// FetchItemByFilter performs a GET request using a query filter.
func (o *exampleOutbound) FetchItemByFilter(ctx context.Context, filter string) ([]models.Item, error) {
	path := "/items"
	if filter != "" {
		path += "?filter=" + url.QueryEscape(filter)
	}

	req := transport.HTTPOutbound{
		Host:     config.Cfg.Service.Example.Host,
		Path:     path,
		Method:   http.MethodGet,
		Headers:  o.headers(ctx),
		Response: &response.GenericResponse{},
	}

	resp, code := req.SendHTTPRequest(o.log)
	if code != nil {
		return nil, fmt.Errorf("external request failed: %s", code.Message)
	}

	generic, ok := resp.RawResponsePayload.(*response.GenericResponse)
	if !ok {
		return nil, fmt.Errorf("unexpected response type")
	}

	data, err := json.Marshal(generic.Body)
	if err != nil {
		return nil, err
	}

	var items []models.Item
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}
