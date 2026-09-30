package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hibiken/asynq"
)

const TypePushNotification = "notification:push"

type PushNotificationPayload struct {
	ProjectID   string            `json:"project_id"`
	Token       string            `json:"token"`
	Title       string            `json:"title"`
	Body        string            `json:"body"`
	ScheduledAt int64             `json:"scheduled_at"`
	Data        map[string]string `json:"data,omitempty"`
}

func main() {
	defaultRedis := os.Getenv("REDIS_ADDR")
	if defaultRedis == "" {
		defaultRedis = "localhost:6379"
	}

	totalTasks := flag.Int("n", 100, "Total number of push notifications to schedule")
	delayDuration := flag.Duration("delay", 5*time.Second, "Delay duration before task execution (e.g. 5s, 10s, 1m)")
	concurrency := flag.Int("concurrency", 10, "Number of concurrent goroutines for enqueuing")
	redisAddr := flag.String("redis", defaultRedis, "Redis server address (host:port)")
	projectID := flag.String("project", "benchmark-project", "FCM project ID")
	flag.Parse()

	log.Println("==================================================")
	log.Println("     Push Notification Benchmark Producer (CLI)   ")
	log.Println("==================================================")
	log.Printf("Target Redis       : %s\n", *redisAddr)
	log.Printf("Total Tasks        : %d\n", *totalTasks)
	log.Printf("Schedule Delay     : %v\n", *delayDuration)
	log.Printf("Enqueue Concurrency: %d\n", *concurrency)
	log.Printf("FCM Project ID     : %s\n", *projectID)

	client := asynq.NewClient(asynq.RedisClientOpt{Addr: *redisAddr})
	defer client.Close()

	targetExecutionTime := time.Now().Add(*delayDuration)
	scheduledAtMs := targetExecutionTime.UnixMilli()

	log.Printf("Target Execution Time: %s (in %v)\n", targetExecutionTime.Format(time.RFC3339), *delayDuration)
	log.Println("Starting enqueue...")

	startTime := time.Now()

	var enqueuedCount int64
	var errorCount int64

	taskChan := make(chan int, *totalTasks)
	for i := 1; i <= *totalTasks; i++ {
		taskChan <- i
	}
	close(taskChan)

	var wg sync.WaitGroup
	for w := 0; w < *concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for taskID := range taskChan {
				payload := PushNotificationPayload{
					ProjectID:   *projectID,
					Token:       fmt.Sprintf("device_token_%06d", taskID),
					Title:       fmt.Sprintf("Benchmark Notification #%d", taskID),
					Body:        "Ini adalah payload benchmark push notification",
					ScheduledAt: scheduledAtMs,
					Data: map[string]string{
						"task_id": fmt.Sprintf("%d", taskID),
					},
				}

				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
					continue
				}

				task := asynq.NewTask(
					TypePushNotification,
					payloadBytes,
					asynq.ProcessIn(*delayDuration),
					asynq.MaxRetry(3),
				)

				_, err = client.Enqueue(task)
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
					log.Printf(" [!] Failed to enqueue task %d: %v", taskID, err)
				} else {
					atomic.AddInt64(&enqueuedCount, 1)
				}
			}
		}()
	}

	wg.Wait()
	elapsed := time.Since(startTime)

	throughput := float64(enqueuedCount) / elapsed.Seconds()

	log.Println("==================================================")
	log.Println("               Enqueue Summary                    ")
	log.Println("==================================================")
	log.Printf("Successfully Enqueued: %d / %d tasks\n", enqueuedCount, *totalTasks)
	log.Printf("Errors               : %d\n", errorCount)
	log.Printf("Elapsed Time         : %v\n", elapsed)
	log.Printf("Enqueue Throughput   : %.2f tasks/sec\n", throughput)
	log.Printf("Tasks will trigger at: %s\n", targetExecutionTime.Format(time.RFC3339))
	log.Println("==================================================")
}
