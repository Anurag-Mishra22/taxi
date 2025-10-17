package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"github.com/Anurag-Mishra22/taxi/services/trip-service/internal/domain"
	tripTypes "github.com/Anurag-Mishra22/taxi/services/trip-service/pkg/types"
	"github.com/Anurag-Mishra22/taxi/shared/env"
	"github.com/Anurag-Mishra22/taxi/shared/metrics"
	pbd "github.com/Anurag-Mishra22/taxi/shared/proto/driver"
	"github.com/Anurag-Mishra22/taxi/shared/proto/trip"
	"github.com/Anurag-Mishra22/taxi/shared/types"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type service struct {
	repo    domain.TripRepository
	metrics *metrics.Metrics
}

func NewService(repo domain.TripRepository, m *metrics.Metrics) *service {
	return &service{
		repo:    repo,
		metrics: m,
	}
}

func (s *service) CreateTrip(ctx context.Context, fare *domain.RideFareModel) (*domain.TripModel, error) {
	t := &domain.TripModel{
		ID:       primitive.NewObjectID(),
		UserID:   fare.UserID,
		Status:   "pending",
		RideFare: fare,
		Driver:   &trip.TripDriver{},
	}

	trip, err := s.repo.CreateTrip(ctx, t)
	if err == nil && s.metrics != nil {
		s.metrics.RecordTripCreated(fare.PackageSlug, "success")
		s.metrics.ActiveTrips.Inc()
	} else if s.metrics != nil {
		s.metrics.RecordTripCreated(fare.PackageSlug, "failed")
	}

	return trip, err
}

func (s *service) GetRoute(ctx context.Context, pickup, destination *types.Coordinate, useOSRMApi bool) (*tripTypes.OsrmApiResponse, error) {
	start := time.Now()
	defer func() {
		if s.metrics != nil {
			duration := time.Since(start)
			s.metrics.ExternalAPICallDuration.WithLabelValues("osrm").Observe(duration.Seconds())
		}
	}()
	if !useOSRMApi {
		// Return a simple mock response in case we don't want to rely on an external API
		return &tripTypes.OsrmApiResponse{
			Routes: []struct {
				Distance float64 `json:"distance"`
				Duration float64 `json:"duration"`
				Geometry struct {
					Coordinates [][]float64 `json:"coordinates"`
				} `json:"geometry"`
			}{
				{
					Distance: 5.0, // 5km
					Duration: 600, // 10 minutes
					Geometry: struct {
						Coordinates [][]float64 `json:"coordinates"`
					}{
						Coordinates: [][]float64{
							{pickup.Latitude, pickup.Longitude},
							{destination.Latitude, destination.Longitude},
						},
					},
				},
			},
		}, nil
	}

	// or use our self hosted API (check the course lesson: "Preparing for External API Failures")
	baseURL := env.GetString("OSRM_API", "http://router.project-osrm.org")

	url := fmt.Sprintf(
		"%s/route/v1/driving/%f,%f;%f,%f?overview=full&geometries=geojson",
		baseURL,
		pickup.Longitude, pickup.Latitude,
		destination.Longitude, destination.Latitude,
	)

	log.Printf("Started Fetching from OSRM API: URL: %s", url)

	resp, err := http.Get(url)
	if err != nil {
		if s.metrics != nil {
			s.metrics.ExternalAPICallsTotal.WithLabelValues("osrm", "error").Inc()
		}
		return nil, fmt.Errorf("failed to fetch route from OSRM API: %v", err)
	}
	defer resp.Body.Close()

	if s.metrics != nil {
		s.metrics.ExternalAPICallsTotal.WithLabelValues("osrm", "success").Inc()
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read the response: %v", err)
	}

	log.Printf("Got response from OSRM API %s", string(body))

	var routeResp tripTypes.OsrmApiResponse
	if err := json.Unmarshal(body, &routeResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &routeResp, nil
}

func (s *service) EstimatePackagesPriceWithRoute(route *tripTypes.OsrmApiResponse) []*domain.RideFareModel {
	start := time.Now()
	baseFares := getBaseFares()
	estimatedFares := make([]*domain.RideFareModel, len(baseFares))

	for i, f := range baseFares {
		estimatedFares[i] = estimateFareRoute(f, route)
		if s.metrics != nil {
			s.metrics.TripsFareCalculated.WithLabelValues(f.PackageSlug).Inc()
		}
	}

	if s.metrics != nil {
		s.metrics.FareCalculationDuration.Observe(time.Since(start).Seconds())
	}

	return estimatedFares
}

func (s *service) GenerateTripFares(ctx context.Context, rideFares []*domain.RideFareModel, userID string, route *tripTypes.OsrmApiResponse) ([]*domain.RideFareModel, error) {
	fares := make([]*domain.RideFareModel, len(rideFares))

	for i, f := range rideFares {
		id := primitive.NewObjectID()

		fare := &domain.RideFareModel{
			UserID:            userID,
			ID:                id,
			TotalPriceInCents: f.TotalPriceInCents,
			PackageSlug:       f.PackageSlug,
			Route:             route,
		}

		if err := s.repo.SaveRideFare(ctx, fare); err != nil {
			return nil, fmt.Errorf("failed to save trip fare: %w", err)
		}

		fares[i] = fare
	}

	return fares, nil
}

func (s *service) GetAndValidateFare(ctx context.Context, fareID, userID string) (*domain.RideFareModel, error) {
	fare, err := s.repo.GetRideFareByID(ctx, fareID)
	if err != nil {
		return nil, fmt.Errorf("failed to get trip fare: %w", err)
	}

	if fare == nil {
		return nil, fmt.Errorf("fare does not exist")
	}

	// User fare validation (user is owner of this fare?)
	if userID != fare.UserID {
		return nil, fmt.Errorf("fare does not belong to the user")
	}

	return fare, nil
}

func estimateFareRoute(f *domain.RideFareModel, route *tripTypes.OsrmApiResponse) *domain.RideFareModel {
	pricingCfg := tripTypes.DefaultPricingConfig()
	carPackagePrice := f.TotalPriceInCents

	distanceKm := route.Routes[0].Distance
	durationInMinutes := route.Routes[0].Duration

	distanceFare := distanceKm * pricingCfg.PricePerUnitOfDistance
	timeFare := durationInMinutes * pricingCfg.PricingPerMinute
	totalPrice := carPackagePrice + distanceFare + timeFare

	return &domain.RideFareModel{
		TotalPriceInCents: totalPrice,
		PackageSlug:       f.PackageSlug,
	}
}

func getBaseFares() []*domain.RideFareModel {
	return []*domain.RideFareModel{
		{
			PackageSlug:       "suv",
			TotalPriceInCents: 200,
		},
		{
			PackageSlug:       "sedan",
			TotalPriceInCents: 350,
		},
		{
			PackageSlug:       "van",
			TotalPriceInCents: 400,
		},
		{
			PackageSlug:       "luxury",
			TotalPriceInCents: 1000,
		},
	}
}

func (s *service) GetTripByID(ctx context.Context, id string) (*domain.TripModel, error) {
	return s.repo.GetTripByID(ctx, id)
}

func (s *service) UpdateTrip(ctx context.Context, tripID string, status string, driver *pbd.Driver) error {
	err := s.repo.UpdateTrip(ctx, tripID, status, driver)
	if err == nil && status == "payed" && s.metrics != nil {
		s.metrics.ActiveTrips.Dec()
	}
	return err
}
