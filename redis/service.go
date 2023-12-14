package redisService

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

func SetKey(ctx context.Context, redisClient *redis.Client, key string, value interface{}) (err error) {
	// Setting expiry time of 1 hour
	err = redisClient.Set(ctx, key, value, time.Hour).Err()
	return
}

func GetKey(ctx context.Context, redisClient *redis.Client, key string) (val string, err error) {
	val, err = redisClient.Get(ctx, key).Result()
	return
}

func DeleteKey(ctx context.Context, redisClient *redis.Client, key string) (err error) {
	resp := redisClient.Del(ctx, key)
	return resp.Err()
}
