package consumer

import (
	"github.com/tidepool-org/go-common/events"
	"github.com/tidepool-org/mailer/mailer"
	"github.com/tidepool-org/mailer/templates"
	"go.uber.org/zap"
)

const (
	Topic = "emails"
)

func New(logger *zap.SugaredLogger, mailr mailer.Mailer, tmplts templates.Templates, globalVars *templates.GlobalVariables) (events.EventConsumer, error) {
	config := events.NewConfig()
	if err := config.LoadFromEnv(); err != nil {
		return nil, err
	}

	config.KafkaTopic = Topic
	return events.NewFaultTolerantConsumerGroup(config, func() (events.MessageConsumer, error) {
		emailEventHandler, err := NewEmailEventHandler(logger, mailr, tmplts, globalVars)
		if err != nil {
			return nil, err
		}
		handler := events.NewDelegatingEmailEventHandler(emailEventHandler)
		return events.NewCloudEventsMessageHandler([]events.EventHandler{
			handler,
		})
	})
}
