package task

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hibiken/asynq"
)

// Task type constant for push notifications
const (
	TypePushNotification = "notification:push"
)

// PushNotificationPayload merepresentasikan data task yang disimpan di Redis.
type PushNotificationPayload struct {
	ProjectID   string            `json:"project_id"`
	Token       string            `json:"token,omitempty"`
	Topic       string            `json:"topic,omitempty"`
	Condition   string            `json:"condition,omitempty"`
	Title       string            `json:"title"`
	Body        string            `json:"body"`
	ScheduledAt int64             `json:"scheduled_at"` // Epoch milliseconds
	Data        map[string]string `json:"data,omitempty"`
}

// Format payload FCM mock server
type FCMMockNotification struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type FCMMockMessage struct {
	Token        string               `json:"token,omitempty"`
	Topic        string               `json:"topic,omitempty"`
	Condition    string               `json:"condition,omitempty"`
	Data         map[string]string    `json:"data,omitempty"`
	Notification *FCMMockNotification `json:"notification,omitempty"`
}

type FCMMockSendRequest struct {
	Message FCMMockMessage `json:"message"`
}

// NewPushNotificationTask membuat Asynq task baru untuk push notification.
func NewPushNotificationTask(payload PushNotificationPayload) (*asynq.Task, error) {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypePushNotification, bytes), nil
}

// PushNotificationHandler menangani pemrosesan task pengiriman push notification.
type PushNotificationHandler struct {
	client        *http.Client
	mockServerURL string
}

// NewPushNotificationHandler menginisialisasi handler dengan HTTP connection pooling.
func NewPushNotificationHandler(mockServerURL string) *PushNotificationHandler {
	baseURL := strings.TrimRight(mockServerURL, "/")
	return &PushNotificationHandler{
		client: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        500,
				MaxIdleConnsPerHost: 200,
				IdleConnTimeout:     90 * time.Second,
				DisableKeepAlives:   false,
			},
		},
		mockServerURL: baseURL,
	}
}

// ProcessTask mengimplementasikan interface asynq.Handler.
func (h *PushNotificationHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var p PushNotificationPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("failed to unmarshal push notification payload: %w", err)
	}

	if p.ProjectID == "" {
		p.ProjectID = "benchmark-project"
	}

	// Siapkan data map dan sertakan scheduled_at untuk pengukuran jitter di mock server
	dataMap := make(map[string]string)
	for k, v := range p.Data {
		dataMap[k] = v
	}
	if p.ScheduledAt > 0 {
		dataMap["scheduled_at"] = strconv.FormatInt(p.ScheduledAt, 10)
	}

	reqBody := FCMMockSendRequest{
		Message: FCMMockMessage{
			Token:     p.Token,
			Topic:     p.Topic,
			Condition: p.Condition,
			Data:      dataMap,
			Notification: &FCMMockNotification{
				Title: p.Title,
				Body:  p.Body,
			},
		},
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal FCM request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/v1/projects/%s/messages:send", h.mockServerURL, p.ProjectID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(jsonBytes))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		log.Printf(" [!] HTTP POST error to mock server: %v", err)
		return fmt.Errorf("HTTP POST to %s failed: %w", endpoint, err)
	}
	defer resp.Body.Close()

	// Drain body agar TCP connection dapat digunakan kembali oleh pool
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("mock server returned non-2xx status: %d", resp.StatusCode)
	}

	return nil
}
