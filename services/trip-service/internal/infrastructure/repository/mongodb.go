package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/Anurag-Mishra22/taxi/services/trip-service/internal/domain"
	"github.com/Anurag-Mishra22/taxi/shared/db"
	"github.com/Anurag-Mishra22/taxi/shared/metrics"
	pbd "github.com/Anurag-Mishra22/taxi/shared/proto/driver"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type mongoRepository struct {
	db      *mongo.Database
	metrics *metrics.Metrics
}

func NewMongoRepository(db *mongo.Database, m *metrics.Metrics) *mongoRepository {
	return &mongoRepository{
		db:      db,
		metrics: m,
	}
}

func (r *mongoRepository) CreateTrip(ctx context.Context, trip *domain.TripModel) (*domain.TripModel, error) {
	start := time.Now()
	result, err := r.db.Collection(db.TripsCollection).InsertOne(ctx, trip)
	status := "success"
	if err != nil {
		status = "error"
	}
	if r.metrics != nil {
		r.metrics.RecordDBQuery("insert", "trips", status, time.Since(start))
	}
	if err != nil {
		return nil, err
	}

	trip.ID = result.InsertedID.(primitive.ObjectID)

	return trip, nil
}

func (r *mongoRepository) GetTripByID(ctx context.Context, id string) (*domain.TripModel, error) {
	_id, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	result := r.db.Collection(db.TripsCollection).FindOne(ctx, bson.M{"_id": _id})
	status := "success"
	if result.Err() != nil {
		status = "error"
	}
	if r.metrics != nil {
		r.metrics.RecordDBQuery("find", "trips", status, time.Since(start))
	}
	if result.Err() != nil {
		return nil, result.Err()
	}

	var trip domain.TripModel
	err = result.Decode(&trip)
	if err != nil {
		return nil, err
	}

	return &trip, nil
}

func (r *mongoRepository) UpdateTrip(ctx context.Context, tripID string, status string, driver *pbd.Driver) error {
	_id, err := primitive.ObjectIDFromHex(tripID)
	if err != nil {
		return err
	}

	update := bson.M{"$set": bson.M{"status": status}}

	if driver != nil {
		update["$set"].(bson.M)["driver"] = driver
	}

	start := time.Now()
	result, err := r.db.Collection(db.TripsCollection).UpdateOne(ctx, bson.M{"_id": _id}, update)
	updateStatus := "success"
	if err != nil {
		updateStatus = "error"
	}
	if r.metrics != nil {
		r.metrics.RecordDBQuery("update", "trips", updateStatus, time.Since(start))
	}
	if err != nil {
		return err
	}

	if result.ModifiedCount == 0 {
		return fmt.Errorf("trip not found: %s", tripID)
	}

	return nil
}

func (r *mongoRepository) SaveRideFare(ctx context.Context, fare *domain.RideFareModel) error {
	start := time.Now()
	result, err := r.db.Collection(db.RideFaresCollection).InsertOne(ctx, fare)
	status := "success"
	if err != nil {
		status = "error"
	}
	if r.metrics != nil {
		r.metrics.RecordDBQuery("insert", "ride_fares", status, time.Since(start))
	}
	if err != nil {
		return err
	}

	fare.ID = result.InsertedID.(primitive.ObjectID)

	return nil
}

func (r *mongoRepository) GetRideFareByID(ctx context.Context, id string) (*domain.RideFareModel, error) {
	_id, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	result := r.db.Collection(db.RideFaresCollection).FindOne(ctx, bson.M{"_id": _id})
	status := "success"
	if result.Err() != nil {
		status = "error"
	}
	if r.metrics != nil {
		r.metrics.RecordDBQuery("find", "ride_fares", status, time.Since(start))
	}
	if result.Err() != nil {
		return nil, result.Err()
	}

	var fare domain.RideFareModel
	err = result.Decode(&fare)
	if err != nil {
		return nil, err
	}

	return &fare, nil
}
