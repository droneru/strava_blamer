// Package strava is a minimal Strava API client with refresh-token persistence.
package strava

import (
	"context"
	"time"
)

type Activity struct {
	ID        int64
	Name      string
	SportType string
	DistanceM float64
	StartDate time.Time
}

type Lap struct {
	Index      int
	DistanceM  float64
	MovingTime time.Duration
}

type Client interface {
	GetActivity(ctx context.Context, id int64) (Activity, error)
	GetLaps(ctx context.Context, id int64) ([]Lap, error)
}
