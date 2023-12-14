package users

import (
	"context"
	redisService "shems/redis"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

func GetPasswordHash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func GetStartOfMonth(dateString string) (time.Time, error) {
	// Parse the date string
	date, err := time.Parse("01/02/2006", dateString)
	if err != nil {
		return time.Time{}, err
	}

	// Get the start of the month
	startOfMonth := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, date.Location())

	return startOfMonth, nil
}

func TakeRedisLock(ctx context.Context, redisClient *redis.Client, key string) string {
	// first check if redis lock already exists
	val, err := redisService.GetKey(ctx, redisClient, key)

	// redis.Nil error indicates that key is not present in redis
	if err != nil && err != redis.Nil {
		return "error while getting key from redis"
	}

	// parse redis value
	if len(val) > 0 {
		boolVal, err := strconv.ParseBool(val)
		if err != nil {
			return "error while parsing redis value"
		}

		// lock already exists
		if boolVal {
			return "redis lock already exists"
		}
	}

	// set key in redis to take lock
	err = redisService.SetKey(ctx, redisClient, key, true)
	if err != nil {
		return "error while setting redis key"
	}
	return ""
}
