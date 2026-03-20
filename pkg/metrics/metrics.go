package metrics

import (
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	TotalRequests      prometheus.Counter
	RequestsPerBackend *prometheus.CounterVec
	ActiveConnections  prometheus.Gauge
	SessionCount       prometheus.Gauge
	BackendHealth      *prometheus.GaugeVec
	ResponseTime       *prometheus.HistogramVec
	RateLimitExceeded  prometheus.Counter
}

var (
	instance *Metrics
	once     sync.Once
)

func NewMetrics() *Metrics {
	once.Do(func() {
		instance = &Metrics{
			TotalRequests: prometheus.NewCounter(prometheus.CounterOpts{
				Name: "lb_total_requests_total",
				Help: "Total number of requests processed by the load balancer",
			}),
			RequestsPerBackend: prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Name: "lb_requests_per_backend_total",
					Help: "Number of requests routed to each backend",
				},
				[]string{"backend_id"},
			),
			ActiveConnections: prometheus.NewGauge(prometheus.GaugeOpts{
				Name: "lb_active_connections",
				Help: "Number of active connections",
			}),
			SessionCount: prometheus.NewGauge(prometheus.GaugeOpts{
				Name: "lb_session_count",
				Help: "Number of active sticky sessions",
			}),
			BackendHealth: prometheus.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: "lb_backend_health",
					Help: "Health status of each backend (1=healthy, 0=unhealthy)",
				},
				[]string{"backend_id"},
			),
			ResponseTime: prometheus.NewHistogramVec(
				prometheus.HistogramOpts{
					Name:    "lb_response_time_seconds",
					Help:    "Response time distribution",
					Buckets: prometheus.DefBuckets,
				},
				[]string{"backend_id"},
			),
			RateLimitExceeded: prometheus.NewCounter(prometheus.CounterOpts{
				Name: "lb_rate_limit_exceeded_total",
				Help: "Total number of requests that exceeded rate limit",
			}),
		}

		prometheus.MustRegister(instance.TotalRequests)
		prometheus.MustRegister(instance.RequestsPerBackend)
		prometheus.MustRegister(instance.ActiveConnections)
		prometheus.MustRegister(instance.SessionCount)
		prometheus.MustRegister(instance.BackendHealth)
		prometheus.MustRegister(instance.ResponseTime)
		prometheus.MustRegister(instance.RateLimitExceeded)
	})

	return instance
}

func (m *Metrics) IncrementTotalRequests() {
	m.TotalRequests.Inc()
}

func (m *Metrics) IncrementBackendRequests(backendID string) {
	m.RequestsPerBackend.WithLabelValues(backendID).Inc()
}

func (m *Metrics) SetActiveConnections(count int64) {
	m.ActiveConnections.Set(float64(count))
}

func (m *Metrics) SetSessionCount(count int) {
	m.SessionCount.Set(float64(count))
}

func (m *Metrics) SetBackendHealth(backendID string, healthy bool) {
	value := float64(0)
	if healthy {
		value = 1
	}
	m.BackendHealth.WithLabelValues(backendID).Set(value)
}

func (m *Metrics) ObserveResponseTime(backendID string, duration float64) {
	m.ResponseTime.WithLabelValues(backendID).Observe(duration)
}

func (m *Metrics) IncrementRateLimitExceeded() {
	m.RateLimitExceeded.Inc()
}

func Handler() http.Handler {
	return promhttp.Handler()
}

type ResponseRecorder struct {
	http.ResponseWriter
	statusCode int
	written    int64
}

func (r *ResponseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *ResponseRecorder) Write(b []byte) (int, error) {
	n, err := r.ResponseWriter.Write(b)
	atomic.AddInt64(&r.written, int64(n))
	return n, err
}
