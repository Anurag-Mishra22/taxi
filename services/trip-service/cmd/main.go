package main

import (
	h "github.com/Anurag-Mishra22/taxi/services/trip-service/internal/http"
	"github.com/Anurag-Mishra22/taxi/services/trip-service/internal/infrastructure/repository"
	"github.com/Anurag-Mishra22/taxi/services/trip-service/internal/service"
	"log"
	"net/http"
)

func main() {
	inmemRepo := repository.NewInmemRepository()
	svc := service.NewService(inmemRepo)
	mux := http.NewServeMux()

	httphandler := h.HttpHandler{Service: svc}

	mux.HandleFunc("POST /preview", httphandler.HandleTripPreview)

	server := &http.Server{
		Addr:    ":8083",
		Handler: mux,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Printf("HTTP server error: %v", err)
	}
}
