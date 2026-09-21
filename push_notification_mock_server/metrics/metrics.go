package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// RequestsTotal menghitung jumlah total request push notification yang diterima mock server.
	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "push_mock_requests_total",
			Help: "Total number of push notification requests received by the mock server",
		},
		[]string{"status", "target_type"},
	)

	// RequestDuration mengukur latensi pemrosesan HTTP request di mock server.
	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "push_mock_request_duration_seconds",
			Help:    "Duration of HTTP request processing in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"handler"},
	)

	// SchedulingDelay mengukur jeda waktu (latency/jitter) antara target jadwal dan waktu penerimaan aktual.
	SchedulingDelay = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "push_mock_scheduling_delay_seconds",
			Help:    "Delay in seconds between scheduled time and actual arrival at mock server",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0},
		},
		[]string{"target_type"},
	)
)
