package repository

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type IdempotencyStore interface {
	HasProcessed(ctx context.Context, eventID string) (bool, error)
	MarkProcessed(ctx context.Context, eventID string, ttl time.Duration) error
}

type redisIdempotencyStore struct {
	client *redis.Client
}

func NewRedisIdempotencyStore(client *redis.Client) IdempotencyStore {
	return &redisIdempotencyStore{client: client}
}

func idempotencyKey(eventID string) string {
	return "processed:" + eventID
}

func (s *redisIdempotencyStore) HasProcessed(ctx context.Context, eventID string) (bool, error) {
	val, err := s.client.Exists(ctx, idempotencyKey(eventID)).Result()
	if err != nil {
		return false, err
	}
	return val > 0, nil
}

func (s *redisIdempotencyStore) MarkProcessed(ctx context.Context, eventID string, ttl time.Duration) error {
	return s.client.Set(ctx, idempotencyKey(eventID), "1", ttl).Err()
}
