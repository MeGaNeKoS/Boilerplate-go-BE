package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	models "project-template/infrastructure/dto/item"
	"project-template/infrastructure/utils"
	"project-template/outbound"
	"project-template/pkg/code"
       repoitem "project-template/repositories/item"
       services "project-template/services/item"
)

type repository interface {
	GetItemRepository() repoitem.Repository
}

func serviceFromContext(ctx context.Context) services.ServiceImpl {
	r, _ := utils.GetRepoCtx(ctx).(repository)
	out, _ := utils.GetOutboundCtx(ctx).(outbound.Impl)
	log := utils.GetLoggerFromContext(ctx)
	if r == nil || out == nil || log == nil {
		return nil
	}
	return services.NewService(r.GetItemRepository(), out.Example().HTTP(), log)
}

func ListItemsHandler(w http.ResponseWriter, r *http.Request) {
	svc := serviceFromContext(r.Context())
	if svc == nil {
		sendResponse(w, utils.GenerateErrorResponse(code.ErrInternalServerError))
		return
	}
	list, codeErr := svc.ListItems(r.Context())
	if codeErr != nil {
		sendResponse(w, utils.GenerateErrorResponse(*codeErr))
		return
	}
	sendResponse(w, utils.GenerateSuccessResponse(list))
}

func CreateItemHandler(w http.ResponseWriter, r *http.Request) {
	svc := serviceFromContext(r.Context())
	if svc == nil {
		sendResponse(w, utils.GenerateErrorResponse(code.ErrInternalServerError))
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		sendResponse(w, utils.GenerateErrorResponse(code.ErrInternalServerError))
		return
	}
	var req models.Item
	if err := json.Unmarshal(body, &req); err != nil {
		sendResponse(w, utils.GenerateErrorResponse(code.ErrPayloadError, err.Error()))
		return
	}
	created, codeErr := svc.CreateItem(r.Context(), req)
	if codeErr != nil {
		sendResponse(w, utils.GenerateErrorResponse(*codeErr))
		return
	}
	sendResponse(w, utils.GenerateSuccessResponse(created, nil))
}

func GetItemHandler(w http.ResponseWriter, r *http.Request) {
	svc := serviceFromContext(r.Context())
	if svc == nil {
		sendResponse(w, utils.GenerateErrorResponse(code.ErrInternalServerError))
		return
	}
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		sendResponse(w, utils.GenerateErrorResponse(code.ErrBadRequest))
		return
	}
	itm, codeErr := svc.GetItem(r.Context(), id)
	if codeErr != nil {
		sendResponse(w, utils.GenerateErrorResponse(*codeErr))
		return
	}
	sendResponse(w, utils.GenerateSuccessResponse(itm))
}

func UpdateItemHandler(w http.ResponseWriter, r *http.Request) {
	svc := serviceFromContext(r.Context())
	if svc == nil {
		sendResponse(w, utils.GenerateErrorResponse(code.ErrInternalServerError))
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		sendResponse(w, utils.GenerateErrorResponse(code.ErrInternalServerError))
		return
	}
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		sendResponse(w, utils.GenerateErrorResponse(code.ErrBadRequest))
		return
	}
	var req models.Item
	if err := json.Unmarshal(body, &req); err != nil {
		sendResponse(w, utils.GenerateErrorResponse(code.ErrPayloadError, err.Error()))
		return
	}
	req.ID = id
	updated, codeErr := svc.UpdateItem(r.Context(), req)
	if codeErr != nil {
		sendResponse(w, utils.GenerateErrorResponse(*codeErr))
		return
	}
	sendResponse(w, utils.GenerateSuccessResponse(updated, nil))
}

func DeleteItemHandler(w http.ResponseWriter, r *http.Request) {
	svc := serviceFromContext(r.Context())
	if svc == nil {
		sendResponse(w, utils.GenerateErrorResponse(code.ErrInternalServerError))
		return
	}
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		sendResponse(w, utils.GenerateErrorResponse(code.ErrBadRequest))
		return
	}
	if codeErr := svc.DeleteItem(r.Context(), id); codeErr != nil {
		sendResponse(w, utils.GenerateErrorResponse(*codeErr))
		return
	}
	sendResponse(w, utils.GenerateSuccessResponse(nil))
}
