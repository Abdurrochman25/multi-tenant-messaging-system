package services

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type MonitoringService struct {
	MessagesSent        prometheus.Counter
	MessagesProcessed   prometheus.Counter
	MessagesFailed      prometheus.Counter
	QueueDepth          *prometheus.GaugeVec
	WorkerActivity      *prometheus.GaugeVec
	ProcessingDuration  *prometheus.HistogramVec
}

func NewMonitoringService() *MonitoringService {
	return &MonitoringService{
		MessagesSent: promauto.NewCounter(prometheus.CounterOpts{
			Name: "messages_sent_total",
			Help: "The total number of messages sent to queues",
		}),
		MessagesProcessed: promauto.NewCounter(prometheus.CounterOpts{
			Name: "messages_processed_total",
			Help: "The total number of successfully processed messages",
		}),
		MessagesFailed: promauto.NewCounter(prometheus.CounterOpts{
			Name: "messages_failed_total",
			Help: "The total number of failed message processing attempts",
		}),
		QueueDepth: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "queue_depth",
			Help: "Current depth of tenant queues",
		}, []string{"tenant_id"}),
		WorkerActivity: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "worker_activity",
			Help: "Number of active workers per tenant",
		}, []string{"tenant_id"}),
		ProcessingDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "message_processing_duration_seconds",
			Help:    "Duration of message processing",
			Buckets: prometheus.DefBuckets,
		}, []string{"tenant_id", "message_type"}),
	}
}

func (m *MonitoringService) RecordMessageSent() {
	m.MessagesSent.Inc()
}

func (m *MonitoringService) RecordMessageProcessed(tenantID, messageType string, duration float64) {
	m.MessagesProcessed.Inc()
	m.ProcessingDuration.WithLabelValues(tenantID, messageType).Observe(duration)
}

func (m *MonitoringService) RecordMessageFailed() {
	m.MessagesFailed.Inc()
}

func (m *MonitoringService) UpdateQueueDepth(tenantID string, depth float64) {
	m.QueueDepth.WithLabelValues(tenantID).Set(depth)
}

func (m *MonitoringService) UpdateWorkerActivity(tenantID string, activeWorkers float64) {
	m.WorkerActivity.WithLabelValues(tenantID).Set(activeWorkers)
}