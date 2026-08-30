package repositories

import (
	"context"
	"time"
)

type ProcessedEventCleaner interface {
	DeleteProcessedBefore(
		ctx context.Context,
		cutoff time.Time,
	) (int64, error)
}
