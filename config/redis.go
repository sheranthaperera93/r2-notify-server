package config

import (
	"context"
	"crypto/tls"
	"log"
	"strconv"

	"github.com/redis/go-redis/v9"
)

var (
	RDB *redis.Client
	Ctx = context.Background()
)

func InitRedis() {
	redisHost := LoadConfig().RedisHost
	redisPort := LoadConfig().RedisPort
	RedisPassword := LoadConfig().RedisPassword
	RDB = redis.NewClient(&redis.Options{
		Addr:      redisHost + ":" + strconv.Itoa(redisPort),
		Password:  RedisPassword,
		TLSConfig: &tls.Config{MinVersion: tls.VersionTLS12},
	})

	if _, err := RDB.Ping(Ctx).Result(); err != nil {
		log.Fatalf("Redis connection failed: %v", err)
		panic(err)
	}

	log.Println("Connected to Redis")
}
