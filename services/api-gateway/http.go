package main

import (
	"encoding/json"
	"github.com/Anurag-Mishra22/taxi/shared/contracts"
	"net/http"
)

func handleTripPreview(w http.ResponseWriter, r *http.Request) {

	var reqBody previewTripRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "failed to parse JSON data", http.StatusBadRequest)
		return
	}

	// validation
	if reqBody.UserID == "" {
		http.Error(w, "user ID is required", http.StatusBadRequest)
		return
	}

	// TODO: Call trip service

	response := contracts.APIResponse{Data: "ok"}

	writeJSON(w, http.StatusCreated, response)

}
