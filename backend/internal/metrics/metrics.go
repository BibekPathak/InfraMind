package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

// Metrics holds the Prometheus collectors. Constructed once in main and
// injected into components — no global state.
type Metrics struct {
	reg *prometheus.Registry
	HTTPRequests   *prometheus.CounterVec
	HTTPDuration   *prometheus.HistogramVec
	TelemetryIn    prometheus.Counter
	TelemetryBatch prometheus.Histogram
	MQTTMessages   prometheus.Counter
	AlertsRaised   prometheus.Counter
	ActionsExecuted prometheus.Counter
	EventBusPublish prometheus.Counter
}

func New(reg *prometheus.Registry) *Metrics {
	m := &Metrics{
		reg: reg,
		HTTPRequests: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "infra_http_requests_total",
				Help: "Total HTTP requests by method, path and status",
			},
			[]string{"method", "path", "status"},
		),
		HTTPDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "infra_http_request_duration_seconds",
				Help:    "HTTP request latency",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
		TelemetryIn: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "infra_telemetry_ingested_total",
				Help: "Total telemetry points ingested",
			},
		),
		TelemetryBatch: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "infra_telemetry_batch_size",
				Help:    "Telemetry batch sizes flushed to DB",
				Buckets: prometheus.ExponentialBuckets(10, 4, 6),
			},
		),
		MQTTMessages: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "infra_mqtt_messages_total",
				Help: "Total MQTT messages received",
			},
		),
		AlertsRaised: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "infra_alerts_raised_total",
				Help: "Total alerts raised",
			},
		),
		ActionsExecuted: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "infra_actions_executed_total",
				Help: "Total autonomous actions executed",
			},
		),
		EventBusPublish: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "infra_eventbus_publish_total",
				Help: "Total events published to the event bus",
			},
		),
	}

	reg.MustRegister(
		m.HTTPRequests, m.HTTPDuration, m.TelemetryIn, m.TelemetryBatch,
		m.MQTTMessages, m.AlertsRaised, m.ActionsExecuted, m.EventBusPublish,
	)
	return m
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.reg, promhttp.HandlerOpts{})
}

func (m *Metrics) ObserveHTTP(method, path string, status int, dur time.Duration) {
	m.HTTPRequests.WithLabelValues(method, path, fmtStatus(status)).Inc()
	m.HTTPDuration.WithLabelValues(method, path).Observe(dur.Seconds())
}

func (m *Metrics) IncTelemetryIngested() {
	m.TelemetryIn.Inc()
}

func (m *Metrics) ObserveBatchSize(size int) {
	m.TelemetryBatch.Observe(float64(size))
}

func (m *Metrics) IncMQTTMessage() {
	m.MQTTMessages.Inc()
}

func (m *Metrics) IncAlertsRaised() {
	m.AlertsRaised.Inc()
}

func (m *Metrics) IncActionsExecuted() {
	m.ActionsExecuted.Inc()
}

func (m *Metrics) IncEventBusPublish() {
	m.EventBusPublish.Inc()
}

func fmtStatus(code int) string {
	switch {
	case code >= 500:
		return "5xx"
	case code >= 400:
		return "4xx"
	default:
		return "2xx"
	}
}
