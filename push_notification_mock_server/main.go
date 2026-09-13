package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	r.POST("/v1/projects/:project_id/messages:send", sendFCMMessage)

	r.Run()
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
	projectID := c.Param("project_id")

	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, GoogleErrorResponse{
			Error: GoogleRPCError{
				Code:    http.StatusBadRequest,
				Message: fmt.Sprintf("Invalid JSON payload: %v", err.Error()),
				Status:  "INVALID_ARGUMENT",
			},
		})
		return
	}

	if req.Message.Token == "" && req.Message.Topic == "" && req.Message.Condition == "" {
		c.JSON(http.StatusBadRequest, GoogleErrorResponse{
			Error: GoogleRPCError{
				Code:    http.StatusBadRequest,
				Message: "Request must specify a target: 'token', 'topic', or 'condition'",
				Status:  "INVALID_ARGUMENT",
			},
		})
		return
	}

	messageID := fmt.Sprintf("%d", time.Now().UnixNano())
	c.JSON(http.StatusOK, gin.H{
		"name": fmt.Sprintf("projects/%s/messages/%s", projectID, messageID),
	})
}
