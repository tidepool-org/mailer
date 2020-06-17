package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	kfka "github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/tidepool-org/mailer/mailer"
	"go.uber.org/zap"
)

type EmailProducer struct {
	cfg    *Config
	logger *zap.SugaredLogger
	producer *kfka.Producer
}

var _ mailer.Mailer = &EmailProducer{}

func NewEmailProducer(cfg *Config, logger *zap.SugaredLogger) (*EmailProducer, error) {
	producer, err := kfka.NewProducer(&kfka.ConfigMap{"bootstrap.servers": cfg.KafkaBrokers})
	if err != nil {
		return nil, err
	}

	return &EmailProducer{
		cfg:    cfg,
		logger: logger,
		producer: producer,
	}, nil
}

func (k *EmailProducer) Close(timeoutMs int) (err error) {
	outstandingEvents := k.producer.Flush(timeoutMs)
	if outstandingEvents != 0 {
		err = errors.New(fmt.Sprintf("%v events were not delivered", outstandingEvents))
	}
	k.producer.Close()
	return
}

func (k *EmailProducer) Send(ctx context.Context, email *mailer.Email) error {
	b, err := json.Marshal(email)
	if err != nil {
		return err
	}

	deliveryChan := make(chan kfka.Event)
	err = k.producer.Produce(&kfka.Message{
		TopicPartition: kfka.TopicPartition{Topic: &k.cfg.KafkaTopic, Partition: kfka.PartitionAny},
		Value:          b,
	}, deliveryChan)
	if err != nil {
		k.logger.Errorw(
			"Error enqueueing message",
			"topic", k.cfg.KafkaTopic,
			"error", err,
		)
		return err
	}

	e := <-deliveryChan
	m := e.(*kfka.Message)

	if m.TopicPartition.Error != nil {
		k.logger.Errorw(
			"Message delivery failed",
			"topic", m.TopicPartition.Topic,
			"error", m.TopicPartition.Error,
		)
	} else {
		k.logger.Debugw(
			"Successfully delivered message",
			"topic", m.TopicPartition.Topic,
			"partition", m.TopicPartition.Partition,
			"offset", m.TopicPartition.Offset,
		)
	}

	close(deliveryChan)
	return m.TopicPartition.Error
}
