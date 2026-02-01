package config

import (
	"context"
	"crypto/tls"
	"log"
	"strconv"
	"time"

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
	redisTLSEnabled := LoadConfig().RedisTLSEnabled
	log.Printf("Redis Configurations: host=%s, port=%d, username=%s, password=***, tlsEnabled=%s", redisHost, redisPort, redisUsername, redisTLSEnabled)

	options := &redis.Options{
		Addr:         redisHost + ":" + strconv.Itoa(redisPort),
		Username:     redisUsername,
		Password:     redisPassword,
		DB:           0,
		DialTimeout:  10 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     20,
		MinIdleConns: 5,
	}

	if redisTLSEnabled == "true" {
		options.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
		log.Println("TLS enabled for Redis connection")
	} else {
		log.Println("TLS disabled for Redis connection")
	}

	RDB = redis.NewClient(options)

	ctx, cancel := context.WithTimeout(Ctx, 10*time.Second)
	defer cancel()

	if _, err := RDB.Ping(ctx).Result(); err != nil {
		log.Fatalf("Redis connection failed: %v", err)
	}

	log.Printf("Connected to Redis successfully!")
}
