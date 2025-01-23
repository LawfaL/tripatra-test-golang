package config

import (
	"fmt"

	"github.com/go-redis/redis/v8"
)

func ConnectRedis(config *Config) *redis.Client {
	RedisClient := redis.NewClient(&redis.Options{
		Addr: config.RedisUri,
	})

	if _, err := RedisClient.Ping(ctx).Result(); err != nil {
		panic(err)
	}

	err := RedisClient.Set(ctx, "test", "Redis engine on", 0).Err()
	if err != nil {
		panic(err)
	}

	fmt.Println("Redis client connected successfully...")
	return RedisClient
}
