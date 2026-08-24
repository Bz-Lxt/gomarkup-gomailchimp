package limiter

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/redis/go-redis/v9"
)

//go:embed token_bucket.lua
var luaScript string

// RedisBucket is the distributed limiter. Clock source is Redis TIME.
type RedisBucket struct {
	rdb  *redis.Client
	sha  string
	script *redis.Script
}

func NewRedis(rdb *redis.Client) *RedisBucket {
	return &RedisBucket{rdb: rdb, script: redis.NewScript(luaScript)}
}

func (b *RedisBucket) Allow(ctx context.Context, key string, ratePerMin float64, burst float64) (bool, float64, error) {
	if ratePerMin <= 0 {
		return true, burst, nil
	}
	if burst < 1 {
		burst = 1
	}
	res, err := b.script.Run(ctx, b.rdb, []string{"rl:" + key}, ratePerMin, burst).Result()
	if err != nil {
		return false, 0, fmt.Errorf("token bucket: %w", err)
	}
	arr, ok := res.([]any)
	if !ok || len(arr) < 2 {
		return false, 0, fmt.Errorf("token bucket: unexpected result %T", res)
	}
	allowed := toInt(arr[0]) == 1
	return allowed, toFloat(arr[1]), nil
}

func toInt(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	default:
		return 0
	}
}

func toFloat(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int64:
		return float64(n)
	case int:
		return float64(n)
	default:
		return 0
	}
}
