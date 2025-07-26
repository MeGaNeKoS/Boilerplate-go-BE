package handlers

import (
	"encoding/json"
	"net/http"

	"project-template/infrastructure/dto/response"
)

// sendResponse writes the standardized HTTP response payload.
func sendResponse(w http.ResponseWriter, resp *response.HttpResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.HTTPCode)
	if resp.RawResponsePayload != nil {
		_ = json.NewEncoder(w).Encode(resp.RawResponsePayload)
	}
}
