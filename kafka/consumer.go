package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	kfka "github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/tidepool-org/mailer/mailer"
	"go.uber.org/zap"
	"strings"
)

type EmailConsumer struct {
	cfg    *ConsumerConfig
	consumer *kfka.Consumer
	logger *zap.SugaredLogger
	mailer mailer.Mailer
}

func NewEmailConsumer(cfg *ConsumerConfig, logger *zap.SugaredLogger, mailer mailer.Mailer) (*EmailConsumer, error) {
	consumer, err := kfka.NewConsumer(&kfka.ConfigMap{
		"bootstrap.servers": strings.Join(cfg.KafkaBrokers, ","),
		"group.id": cfg.ConsumerGroup,
		"enable.auto.commit": false,
	})
	if err != nil {
		return nil, err
	}

	err = consumer.SubscribeTopics([]string{cfg.KafkaTopic}, nil)
	if err != nil {
		return nil, err
	}

	return &EmailConsumer{
		cfg:    cfg,
		consumer: consumer,
		logger: logger,
		mailer: mailer,
	}, nil
}

// ProcessMessages starts the main loop which fetches email messages from Kafka and sends them
// using the configured backend. All invalid messages are ignored. Valid ones are acknowledged
// when sending an email succeeds.
//
// The call to this function is blocking and cancelling context can be used for graceful shutdown.
func (e *EmailConsumer) ProcessMessages(ctx context.Context) error {
	run := true
	for run == true {
		select {
		case <-ctx.Done():
			e.logger.Debugw(
				"Email consumer context was terminated. Shutting down consumer.",
				"reason", ctx.Err(),
			)
			run = false
		default:
			ev := e.consumer.Poll(int(e.cfg.KafkaPollInterval.Milliseconds()))
			if ev == nil {
				continue
			}

			switch event := ev.(type) {
			case *kfka.Message:
				err := e.sendEmail(event)
				if err == nil {
					// Committing after sending the email ensures 'at-least-once' delivery
					_, err := e.consumer.CommitMessage(event)
					if err != nil {
						ObserveError(CommitFailed, NullErrorCode)
						e.logger.Errorw("Unable to commit message", "error", err)
					}
				}
			case kfka.Error:
				ObserveError(KafkaError, event.Code().String())
				if event.IsFatal() {
					e.logger.Error(event.String())
					return event
				}

				// the client will try to recover automatically
				e.logger.Warnw(event.Code().String())
			default:
				e.logger.Debugw("Ignored event", "event", fmt.Sprintf("%v", event))
			}
		}
	}

	return e.consumer.Close()
}

func (e *EmailConsumer) sendEmail(message *kfka.Message) error {
	email := &mailer.Email{}
	if err := json.Unmarshal(message.Value, email); err != nil {
		ObserveError(InvalidMessage, NullErrorCode)
		e.logger.Errorw(
			"Unable to unmarshal email from kafka message",
			"topic", message.TopicPartition.Topic,
			"partition", message.TopicPartition.Partition,
			"key", message.Key,
			"offset", message.TopicPartition.Offset,
			"error", err,
		)

		// Nothing else we can do, just ignore this message.
		return nil
	}

	if err := e.mailer.Send(context.Background(), email); err != nil {
		e.logger.Errorw("Error sending email", "error", err)
		return err
	}

	e.logger.Debugw(
		"Successfully sent email message",
		"topic", message.TopicPartition.Topic,
		"partition", message.TopicPartition.Partition,
		"key", message.Key,
		"offset", message.TopicPartition.Offset,
	)
	return nil
}