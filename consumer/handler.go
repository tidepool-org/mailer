package consumer

import (
	"context"
	"github.com/tidepool-org/go-common/events"
	"github.com/tidepool-org/mailer/mailer"
	"github.com/tidepool-org/mailer/templates"
	"go.uber.org/zap"
	"time"
)

const timeout = time.Second * 30

type EmailEventHandler struct {
	logger *zap.SugaredLogger
	mailer mailer.Mailer
	tmplts templates.Templates
}

var _ events.EmailEventHandler = &EmailEventHandler{}

func NewEmailEventHandler(logger *zap.SugaredLogger, mailer mailer.Mailer, tmplts templates.Templates) (*EmailEventHandler, error) {
	return &EmailEventHandler{
		logger: logger,
		mailer: mailer,
		tmplts: tmplts,
	}, nil
}

func (e *EmailEventHandler) HandleSendEmailTemplate(payload events.SendEmailTemplateEvent) error {
	tmplt, ok := e.tmplts[templates.TemplateName(payload.Template)]
	if !ok {
		e.logger.Info("Skipping email to %s because template %s doesn't exist", payload.Recipient, payload.Template)
		return nil
	}

	vars := payload.Variables
	rendered, err := tmplt.Execute(vars)
	if err != nil {
		return err
	}

	email := &mailer.Email{
		Recipients: []string{payload.Recipient},
		Subject:    rendered.Subject,
		Body:       rendered.Body,
	}

	ctx, _ := context.WithTimeout(context.Background(), timeout)
	return e.mailer.Send(ctx, email)
}
