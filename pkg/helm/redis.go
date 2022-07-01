package helm

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
)

// redis client single instance
var redis_client *redis.Client = nil

func getRedisClient() (*redis.Client, error) {
	if redis_client != nil {
		return redis_client, nil
	}

	var host string = os.Getenv("ARGOCD_REDIS_HA_HAPROXY_SERVICE_HOST")
	var port string = os.Getenv("ARGOCD_REDIS_HA_HAPROXY_SERVICE_PORT")

	if len(host) == 0 {
		host = os.Getenv("ARGOCD_REDIS_SERVICE_HOST")
		port = os.Getenv("ARGOCD_REDIS_SERVICE_PORT")
	}

	if len(host) == 0 || len(port) == 0 {
		return nil, errors.New("redis host or port environment variable doesn't exist")
	}

	redis_client = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", host, port),
		Password: "",
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key, err := redis_client.Echo(ctx, "test message").Result()
	if err != nil {
		return nil, err
	}

	if key != "test message" {
		return nil, fmt.Errorf("Connect to redis server %s:%s fail", host, port)
	} else {
		log.Printf("Redis server %s:%s connected\n", host, port)
	}

	return redis_client, nil
}
