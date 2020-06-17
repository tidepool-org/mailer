package kafka

import "time"

type Config struct {
	KafkaBrokers      []string      `envconfig:"TIDEPOOL_KAFKA_BROKERS" validate:"required"`
	KafkaTopic        string        `envconfig:"TIDEPOOL_KAFKA_EMAILS_TOPIC" validate:"required"`
}

type ConsumerConfig struct {
	Config
	KafkaPollInterval time.Duration `envconfig:"TIDEPOOL_KAFKA_EMAILS_POOL_INTERVAL" default:"100ms" validate:"required"`
	ConsumerGroup string `envconfig:"TIDEPOOL_KAFKA_CONSUMER_GROUP" default:"mailer" validate:"required"`
}
