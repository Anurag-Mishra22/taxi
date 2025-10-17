package main

import (
	"context"
	"fmt"
	"log"
	math "math/rand/v2"
	"github.com/Anurag-Mishra22/taxi/shared/cache"
	"github.com/Anurag-Mishra22/taxi/shared/metrics"
	pb "github.com/Anurag-Mishra22/taxi/shared/proto/driver"
	"github.com/Anurag-Mishra22/taxi/shared/util"
	"sync"
	"time"

	"github.com/mmcloughlin/geohash"
)

type driverInMap struct {
	Driver *pb.Driver
	// Index int
	// TODO: route
}

type Service struct {
	drivers []*driverInMap
	mu      sync.RWMutex
	metrics *metrics.Metrics
	redis   *cache.RedisClient
}

const (
	RedisDriversOnlineKey    = "drivers:online"    // Redis SET to track online driver IDs
	RedisDriverDataPrefix    = "driver:data:"      // Hash key prefix for driver details
	RedisDriversByPackageKey = "drivers:online:%s" // Redis SET for drivers by package type
)

func NewService(m *metrics.Metrics) *Service {
	// Initialize Redis client
	redisClient, err := cache.NewRedisClient()
	if err != nil {
		// Log error but continue - service will work in degraded mode
		// (only in-memory tracking)
		panic(err) // For production, you may want graceful fallback
	}

	svc := &Service{
		drivers: make([]*driverInMap, 0),
		metrics: m,
		redis:   redisClient,
	}

	// Initial sync with Redis to set correct metric value
	svc.updateDriverMetrics()

	// Start periodic sync every 10 seconds to keep metrics accurate
	go svc.startMetricSyncLoop()

	return svc
}

// FindAvailableDrivers returns IDs of online drivers matching the package type.
// Uses Redis per-package sets for cluster-wide matching across all pods.
// Falls back to in-memory search if Redis is unavailable.
func (s *Service) FindAvailableDrivers(packageType string) []string {
	// Try Redis first (cluster-wide matching)
	if s.redis != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		// Direct lookup from per-package set (O(1) operation)
		packageKey := fmt.Sprintf(RedisDriversByPackageKey, packageType)
		driverIDs, err := s.redis.SMembers(ctx, packageKey)
		if err != nil {
			log.Printf("Failed to get drivers from Redis for package %s: %v, falling back to memory", packageType, err)
			// Fall through to memory-based search
		} else {
			log.Printf("Found %d drivers for package %s from Redis (cluster-wide)", len(driverIDs), packageType)
			return driverIDs
		}
	}

	// Fallback: search in-memory (single-pod only)
	return s.findAvailableDriversInMemory(packageType)
}

// findAvailableDriversInMemory searches only this pod's in-memory driver list.
// Used as fallback when Redis is unavailable.
func (s *Service) findAvailableDriversInMemory(packageType string) []string {
	var matchingDrivers []string

	for _, driver := range s.drivers {
		if driver.Driver.PackageSlug == packageType {
			matchingDrivers = append(matchingDrivers, driver.Driver.Id)
		}
	}

	if len(matchingDrivers) == 0 {
		return []string{}
	}

	log.Printf("Found %d drivers for package %s from memory (single-pod)", len(matchingDrivers), packageType)
	return matchingDrivers
}

func (s *Service) RegisterDriver(driverId string, packageSlug string) (*pb.Driver, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.metrics != nil {
		s.metrics.DriversRegisteredTotal.Inc()
	}

	randomIndex := math.IntN(len(PredefinedRoutes))
	randomRoute := PredefinedRoutes[randomIndex]

	randomPlate := GenerateRandomPlate()
	randomAvatar := util.GetRandomAvatar(randomIndex)

	// we can ignore this property for now, but it must be sent to the frontend.
	geohash := geohash.Encode(randomRoute[0][0], randomRoute[0][1])

	driver := &pb.Driver{
		Id:             driverId,
		Geohash:        geohash,
		Location:       &pb.Location{Latitude: randomRoute[0][0], Longitude: randomRoute[0][1]},
		Name:           "Lando Norris",
		PackageSlug:    packageSlug,
		ProfilePicture: randomAvatar,
		CarPlate:       randomPlate,
	}

	s.drivers = append(s.drivers, &driverInMap{
		Driver: driver,
	})

	// Store driver in Redis
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if s.redis != nil {
		// 1. Add driver ID to global online drivers set
		if err := s.redis.SAdd(ctx, RedisDriversOnlineKey, driverId); err != nil {
			log.Printf("Failed to add driver %s to online set: %v", driverId, err)
			return driver, nil // Still return driver, Redis failure shouldn't block registration
		}

		// 2. Add driver to per-package set for fast matching
		packageKey := fmt.Sprintf(RedisDriversByPackageKey, packageSlug)
		if err := s.redis.SAdd(ctx, packageKey, driverId); err != nil {
			log.Printf("Failed to add driver %s to package set %s: %v", driverId, packageKey, err)
		}
		// Set TTL on package set (slightly longer than driver data to avoid race conditions)
		s.redis.Expire(ctx, packageKey, 31*time.Minute)

		// 3. Store full driver profile for cross-pod access
		driverKey := RedisDriverDataPrefix + driverId
		if err := s.redis.HSetJSON(ctx, driverKey, "data", driver); err != nil {
			log.Printf("Failed to store driver %s data: %v", driverId, err)
		}
		s.redis.Expire(ctx, driverKey, 30*time.Minute)

		log.Printf("Driver %s registered in Redis (global + package:%s)", driverId, packageSlug)
	}

	// Update metrics based on Redis count
	if s.metrics != nil {
		s.updateDriverMetrics()
	}

	return driver, nil
}

func (s *Service) UnregisterDriver(driverId string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove from in-memory list and track package type for Redis cleanup
	var driverPackage string
	for i, driver := range s.drivers {
		if driver.Driver.Id == driverId {
			driverPackage = driver.Driver.PackageSlug
			s.drivers = append(s.drivers[:i], s.drivers[i+1:]...)
			break
		}
	}

	// Remove driver from Redis (even if not in memory - handles pod restarts)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if s.redis != nil {
		// 1. Remove from global online drivers set
		if err := s.redis.SRem(ctx, RedisDriversOnlineKey, driverId); err != nil {
			log.Printf("Failed to remove driver %s from Redis SET: %v", driverId, err)
		} else {
			log.Printf("Removed driver %s from Redis online set", driverId)
		}

		// 2. Remove from per-package set (if we know the package)
		if driverPackage != "" {
			packageKey := fmt.Sprintf(RedisDriversByPackageKey, driverPackage)
			if err := s.redis.SRem(ctx, packageKey, driverId); err != nil {
				log.Printf("Failed to remove driver %s from package set %s: %v", driverId, packageKey, err)
			}
		} else {
			// If package unknown, try to get it from Redis driver data before deleting
			driverKey := RedisDriverDataPrefix + driverId
			var driver pb.Driver
			if err := s.redis.HGetJSON(ctx, driverKey, "data", &driver); err == nil {
				packageKey := fmt.Sprintf(RedisDriversByPackageKey, driver.PackageSlug)
				s.redis.SRem(ctx, packageKey, driverId)
			}
		}

		// 3. Delete driver profile data
		driverKey := RedisDriverDataPrefix + driverId
		if err := s.redis.Del(ctx, driverKey); err != nil {
			log.Printf("Failed to delete driver %s data from Redis: %v", driverId, err)
		}

		log.Printf("Driver %s fully unregistered from Redis", driverId)
	}

	// Update metrics based on Redis count
	if s.metrics != nil {
		s.updateDriverMetrics()
	}
}

// startMetricSyncLoop periodically syncs metrics with Redis state
func (s *Service) startMetricSyncLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.updateDriverMetrics()
	}
}

// updateDriverMetrics synchronizes the DriversOnline metric with Redis
func (s *Service) updateDriverMetrics() {
	if s.redis == nil || s.metrics == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Get count from Redis
	count, err := s.redis.SCard(ctx, RedisDriversOnlineKey)
	if err != nil {
		return
	}

	// Set the gauge to the exact Redis count
	s.metrics.DriversOnline.Set(float64(count))
}
