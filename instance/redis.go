package instance

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

type Redis interface {
	Subscribe(ctx context.Context, ch chan string, subscribeTo ...string)
	Ping(ctx context.Context) error
	Publish(ctx context.Context, channel string, content string) error
	Expire(ctx context.Context, key string, ttl time.Duration) error
	Del(ctx context.Context, key string) error
	Get(ctx context.Context, key string) (interface{}, error)
	SetNX(ctx context.Context, key string, value string, ttl time.Duration) (bool, error)
	SetEX(ctx context.Context, key string, value string, ttl time.Duration) error
	Set(ctx context.Context, key string, value string) error
	RawClient() *redis.Client
}
