package repositories

import (
	"context"
	"time"
)

type ProcessedEventRepository interface {
	TryMarkProcessed(
		ctx context.Context,
		consumerName string,
		eventID string,
		subject string,
	) (bool, error)
}

type ProcessedEventCleaner interface {
	DeleteProcessedBefore(
		ctx context.Context,
		cutoff time.Time,
	) (int64, error)
}
