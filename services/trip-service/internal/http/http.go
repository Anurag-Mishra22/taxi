package http

import (
	"encoding/json"
	"github.com/Anurag-Mishra22/taxi/services/trip-service/internal/domain"
	"github.com/Anurag-Mishra22/taxi/shared/types"
	"log"
	"net/http"
)

type HttpHandler struct {
	Service domain.TripService
}

type previewTripRequest struct {
	UserID      string           `json:"userID"`
	Pickup      types.Coordinate `json:"pickup"`
	Destination types.Coordinate `json:"destination"`
}

func (s *HttpHandler) HandleTripPreview(w http.ResponseWriter, r *http.Request) {
	var reqBody previewTripRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "failed to parse JSON data", http.StatusBadRequest)
		return
	}

	fare := &domain.RideFareModel{
		UserID: "42",
	}

	ctx := r.Context()

	t, err := s.Service.CreateTrip(ctx, fare)
	if err != nil {
		log.Println(err)
	}

	writeJSON(w, http.StatusOK, t)
}

func writeJSON(w http.ResponseWriter, status int, data any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}
