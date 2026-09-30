package main

import (
	"log"
	"os"
	"strconv"

	"github.com/faiq1818/skripsi/in_memory_cache/consumer/task"
	"github.com/hibiken/asynq"
)

func main() {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	mockServerURL := os.Getenv("MOCK_SERVER_URL")
	if mockServerURL == "" {
		mockServerURL = "http://localhost:8081"
	}

	concurrency := 10
	if cStr := os.Getenv("CONSUMER_CONCURRENCY"); cStr != "" {
		if c, err := strconv.Atoi(cStr); err == nil && c > 0 {
			concurrency = c
		}
	}

	log.Printf("Starting Push Notification Consumer...")
	log.Printf("Redis Address: %s", redisAddr)
	log.Printf("Mock Server URL: %s", mockServerURL)
	log.Printf("Worker Concurrency: %d", concurrency)

	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr},
		asynq.Config{
			Concurrency: concurrency,
			Queues: map[string]int{
				"default": 1,
			},
		},
	)

	mux := asynq.NewServeMux()
	mux.Handle(task.TypePushNotification, task.NewPushNotificationHandler(mockServerURL))

	if err := srv.Run(mux); err != nil {
		log.Fatalf("Consumer server terminated with error: %v", err)
	}
}
