package mailer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/kafka"
)

type KafkaMailerConfig struct {
	KafkaBrokers      []string `envconfig:"TIDEPOOL_KAFKA_BROKERS" required:"true"`
	KafkaTopic        string   `envconfig:"TIDEPOOL_KAFKA_EMAILS_TOPIC" required:"true"`
}

type KafkaMailer struct {
	cfg          *KafkaMailerConfig
	deliveryChan chan kafka.Event
	producer     *kafka.Producer
}

var _ Mailer = &KafkaMailer{}

func NewKafkaMailer(cfg *KafkaMailerConfig, deliveryChan chan kafka.Event) (*KafkaMailer, error) {
	producer, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": cfg.KafkaBrokers})
	if err != nil {
		return nil, err
	}

	return &KafkaMailer{
		cfg:          cfg,
		deliveryChan: deliveryChan,
		producer:     producer,
	}, nil
}

func (k *KafkaMailer) Send(ctx context.Context, email *Email) error {
	b, err := json.Marshal(email)
	if err != nil {
		return err
	}

	err = k.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &k.cfg.KafkaTopic, Partition: kafka.PartitionAny},
		Value:          b,
	}, k.deliveryChan)
	return err
}

func (k *KafkaMailer) Close(timeoutMs int) (err error) {
	outstandingEvents := k.producer.Flush(timeoutMs)
	if outstandingEvents != 0 {
		err = errors.New(fmt.Sprintf("%v events were not delivered", outstandingEvents))
	}
	k.producer.Close()
	close(k.deliveryChan)
	return
}
