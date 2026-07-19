package weather

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"home-sensors/internal/models"
)

// Cache holds the last fetched forecast in memory and refreshes it hourly
// in the background, so request handlers never call the upstream API
// directly.
type Cache struct {
	lat, lon float64
	log      *slog.Logger

	mu        sync.RWMutex
	points    []models.WeatherPoint
	fetchedAt time.Time
}

func NewCache(lat, lon float64, log *slog.Logger) *Cache {
	return &Cache{lat: lat, lon: lon, log: log}
}

// Start fetches the forecast immediately and then keeps refreshing it every
// hour until ctx is canceled.
func (c *Cache) Start(ctx context.Context) {
	c.refresh(ctx)

	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.refresh(ctx)
			}
		}
	}()
}

func (c *Cache) refresh(ctx context.Context) {
	points, err := FetchToday(ctx, c.lat, c.lon)
	if err != nil {
		c.log.Error("weather refresh failed", "err", err)
		return
	}

	c.mu.Lock()
	c.points = points
	c.fetchedAt = time.Now()
	c.mu.Unlock()
}

// Get returns the cached forecast and when it was fetched. fetchedAt is the
// zero time if no successful fetch has happened yet.
func (c *Cache) Get() ([]models.WeatherPoint, time.Time) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.points, c.fetchedAt
}
