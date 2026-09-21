package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/faiq1818/skripsi/push_notification_mock_server/metrics"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	r.POST("/v1/projects/:project_id/messages:send", sendFCMMessage)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	if err := r.Run(port); err != nil {
		fmt.Printf("Failed to run server: %v\n", err)
	}
}

type Notification struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Image string `json:"image,omitempty"`
}

type AndroidNotification struct {
	Title             string `json:"title,omitempty"`
	Body              string `json:"body,omitempty"`
	Icon              string `json:"icon,omitempty"`
	Color             string `json:"color,omitempty"`
	Sound             string `json:"sound,omitempty"`
	Tag               string `json:"tag,omitempty"`
	ClickAction       string `json:"click_action,omitempty"`
	ChannelID         string `json:"channel_id,omitempty"`
	NotificationCount int    `json:"notification_count,omitempty"`
	Image             string `json:"image,omitempty"`
}

type AndroidConfig struct {
	CollapseKey           string               `json:"collapse_key,omitempty"`
	Priority              string               `json:"priority,omitempty"`
	TTL                   string               `json:"ttl,omitempty"`
	RestrictedPackageName string               `json:"restricted_package_name,omitempty"`
	Data                  map[string]string    `json:"data,omitempty"`
	Notification          *AndroidNotification `json:"notification,omitempty"`
}

type WebpushConfig struct {
	Headers      map[string]string      `json:"headers,omitempty"`
	Data         map[string]string      `json:"data,omitempty"`
	Notification map[string]interface{} `json:"notification,omitempty"`
}

type ApnsConfig struct {
	Headers map[string]string      `json:"headers,omitempty"`
	Payload map[string]interface{} `json:"payload,omitempty"`
}

type Message struct {
	Token        string            `json:"token,omitempty"`
	Topic        string            `json:"topic,omitempty"`
	Condition    string            `json:"condition,omitempty"`
	Data         map[string]string `json:"data,omitempty"`
	Notification *Notification     `json:"notification,omitempty"`
	Android      *AndroidConfig    `json:"android,omitempty"`
	Webpush      *WebpushConfig    `json:"webpush,omitempty"`
	Apns         *ApnsConfig       `json:"apns,omitempty"`
}

type SendMessageRequest struct {
	ValidateOnly bool    `json:"validate_only,omitempty"`
	Message      Message `json:"message" binding:"required"`
}

type GoogleRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

type GoogleErrorResponse struct {
	Error GoogleRPCError `json:"error"`
}

func sendFCMMessage(c *gin.Context) {
	start := time.Now()
	defer func() {
		metrics.RequestDuration.WithLabelValues("send_fcm_message").Observe(time.Since(start).Seconds())
	}()

	projectID := c.Param("project_id")

	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		metrics.RequestsTotal.WithLabelValues("400", "unknown").Inc()
		c.JSON(http.StatusBadRequest, GoogleErrorResponse{
			Error: GoogleRPCError{
				Code:    http.StatusBadRequest,
				Message: fmt.Sprintf("Invalid JSON payload: %v", err.Error()),
				Status:  "INVALID_ARGUMENT",
			},
		})
		return
	}

	targetType := "unknown"
	if req.Message.Token != "" {
		targetType = "token"
	} else if req.Message.Topic != "" {
		targetType = "topic"
	} else if req.Message.Condition != "" {
		targetType = "condition"
	}

	if targetType == "unknown" {
		metrics.RequestsTotal.WithLabelValues("400", targetType).Inc()
		c.JSON(http.StatusBadRequest, GoogleErrorResponse{
			Error: GoogleRPCError{
				Code:    http.StatusBadRequest,
				Message: "Request must specify a target: 'token', 'topic', or 'condition'",
				Status:  "INVALID_ARGUMENT",
			},
		})
		return
	}

	// Hitung scheduling delay jika data 'scheduled_at' disertakan dalam payload
	if req.Message.Data != nil {
		if scheduledStr, exists := req.Message.Data["scheduled_at"]; exists {
			var scheduledTime time.Time
			if ts, err := strconv.ParseInt(scheduledStr, 10, 64); err == nil {
				if ts > 1e12 { // Timestamp milidetik
					scheduledTime = time.UnixMilli(ts)
				} else { // Timestamp detik
					scheduledTime = time.Unix(ts, 0)
				}
			} else if t, err := time.Parse(time.RFC3339Nano, scheduledStr); err == nil {
				scheduledTime = t
			} else if t, err := time.Parse(time.RFC3339, scheduledStr); err == nil {
				scheduledTime = t
			}

			if !scheduledTime.IsZero() {
				delay := time.Since(scheduledTime).Seconds()
				metrics.SchedulingDelay.WithLabelValues(targetType).Observe(delay)
			}
		}
	}

	metrics.RequestsTotal.WithLabelValues("200", targetType).Inc()

	messageID := fmt.Sprintf("%d", time.Now().UnixNano())
	c.JSON(http.StatusOK, gin.H{
		"name": fmt.Sprintf("projects/%s/messages/%s", projectID, messageID),
	})
}
