package client

import (
	"context"
	"github.com/go-playground/validator/v10"
	"github.com/tidepool-org/go-common/events"
)

const Topic = "emails"

type Client interface {
	Send(ctx context.Context, email events.SendEmailTemplateEvent) error
}

type client struct {
	producer *events.KafkaCloudEventsProducer
	validate *validator.Validate
}

var _ Client = &client{}

func NewClient(config *events.CloudEventsConfig) (Client, error) {
	producer, err := events.NewKafkaCloudEventsProducer(config)
	if err != nil {
		return nil, err
	}

	return &client{
		producer: producer,
		validate : validator.New(),
	}, nil
}

func (c *client) Send(ctx context.Context, email events.SendEmailTemplateEvent) error {
	if err := c.validate.Var(email.Recipient, "email"); err != nil {
		return err
	}
	if err := c.validate.Var(email.Template, "required,min=1"); err != nil {
		return err
	}
	return c.producer.Send(ctx, email)
}
