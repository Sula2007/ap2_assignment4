package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"

	"order-service/internal/domain"
)

type OrderCache interface {
	Get(ctx context.Context, id string) (*domain.Order, error)
	Set(ctx context.Context, order *domain.Order, ttl time.Duration) error
	Delete(ctx context.Context, id string) error
}

type redisOrderCache struct {
	client *redis.Client
}

func NewRedisOrderCache(client *redis.Client) OrderCache {
	return &redisOrderCache{client: client}
}

func cacheKey(id string) string {
	return "order:" + id
}

func (c *redisOrderCache) Get(ctx context.Context, id string) (*domain.Order, error) {
	val, err := c.client.Get(ctx, cacheKey(id)).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var order domain.Order
	if err := json.Unmarshal([]byte(val), &order); err != nil {
		return nil, err
	}
	return &order, nil
}

func (c *redisOrderCache) Set(ctx context.Context, order *domain.Order, ttl time.Duration) error {
	data, err := json.Marshal(order)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, cacheKey(order.ID), data, ttl).Err()
}

func (c *redisOrderCache) Delete(ctx context.Context, id string) error {
	return c.client.Del(ctx, cacheKey(id)).Err()
}
