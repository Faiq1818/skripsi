package main

import (
	"encoding/json"
	"log"

	"github.com/hibiken/asynq"
)

type EmailTaskPayload struct {
	// ID for the email recipient.
	UserID int
}

func main() {
	redisConnOpt := asynq.RedisClientOpt{
		Addr: "localhost:6379",
		// Omit if no password is required
		// Password: "mypassword",
		// Use a dedicated db number for asynq.
		// By default, Redis offers 16 databases (0..15)
		DB: 0,
	}

	client := asynq.NewClient(redisConnOpt)

	// Create a task with typename and payload.
	payload, err := json.Marshal(EmailTaskPayload{UserID: 42})
	if err != nil {
		log.Fatal(err)
	}
	t1 := asynq.NewTask("email:welcome", payload)

	info, err := client.Enqueue(t1)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf(" [*] Successfully enqueued task: %+v", info)
}
