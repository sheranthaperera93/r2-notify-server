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
	redisUsername := LoadConfig().RedisUsername
	redisPassword := LoadConfig().RedisPassword
	log.Printf("Redis Configurations: host=%s, port=%d, username=%s, password=***", redisHost, redisPort, redisUsername)
	RDB = redis.NewClient(&redis.Options{
		Addr:      redisHost + ":" + strconv.Itoa(redisPort),
		Username: 	redisUsername,
		Password:  redisPassword,
		TLSConfig: &tls.Config{MinVersion: tls.VersionTLS12},
	})

	if _, err := RDB.Ping(Ctx).Result(); err != nil {
		log.Fatalf("Redis connection failed: %v", err)
		panic(err)
	}

	log.Printf("Connected to Redis")
}
