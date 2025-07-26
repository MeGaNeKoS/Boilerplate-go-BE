package handlers

import (
	"net/http"
	"strconv"

	"project-template/infrastructure/utils"
       svc "project-template/services/system"
)

var systemService = svc.NewService()

// EchoHandler responds with a simple echo message.
func EchoHandler(w http.ResponseWriter, r *http.Request) {
	resp, code := systemService.Echo(r.Context())
	if code != nil {
		sendResponse(w, utils.GenerateErrorResponse(*code))
		return
	}
	sendResponse(w, utils.GenerateSuccessResponse(resp))
}

// CrashHandler intentionally panics to test recovery middleware.
func CrashHandler(w http.ResponseWriter, r *http.Request) {
	systemService.Crash(r.Context())
}

// LongHandler waits for the specified number of seconds before responding.
func LongHandler(w http.ResponseWriter, r *http.Request) {
	sleepStr := r.URL.Query().Get("sleep")
	secs, _ := strconv.Atoi(sleepStr)
	resp, code := systemService.Long(r.Context(), secs)
	if code != nil {
		sendResponse(w, utils.GenerateErrorResponse(*code))
		return
	}
	sendResponse(w, utils.GenerateSuccessResponse(resp))
}
