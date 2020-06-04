package kafka

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	InvalidMessage = "invalid_message"
	CommitFailed = "commit_failed"
	KafkaError = "kafka_error"
	NullErrorCode = "null"
)

var (
	errorCounter = createErrorCounter()
)

func createErrorCounter() *prometheus.CounterVec {
	counter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "tidepool",
			Subsystem: "mailer",
			Name: "kafka_consumer_errors",
		},
		[]string{"label", "code"},
	)

	prometheus.MustRegister(counter)
	return counter
}

func ObserveError(label string, code string) {
	errorCounter.WithLabelValues(label, code).Inc()
}