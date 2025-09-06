package consumer

import (
	"context"
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/tidepool-org/go-common/events"
	"github.com/tidepool-org/mailer/mailer"
	"github.com/tidepool-org/mailer/templates"
	"go.uber.org/zap"
	"time"
)

const timeout = time.Second * 30

type EmailEventHandler struct {
	globalVars *templates.GlobalVariables
	logger     *zap.SugaredLogger
	mailer     mailer.Mailer
	tmplts     templates.Templates
	validate   *validator.Validate
}

var _ events.EmailEventHandler = &EmailEventHandler{}

func NewEmailEventHandler(logger *zap.SugaredLogger, mailer mailer.Mailer, tmplts templates.Templates, globalVars *templates.GlobalVariables) (*EmailEventHandler, error) {
	return &EmailEventHandler{
		globalVars: globalVars,
		logger:     logger,
		mailer:     mailer,
		tmplts:     tmplts,
		validate:   validator.New(),
	}, nil
}

func (e *EmailEventHandler) HandleSendEmailTemplate(payload events.SendEmailTemplateEvent) error {
	tmplt, ok := e.tmplts[templates.TemplateName(payload.Template)]
	if !ok {
		e.logger.Infof("Skipping email to %s because template %s doesn't exist", payload.Recipient, payload.Template)
		return nil
	}

	if err := e.validate.Var(payload.Recipient, "required,email"); err != nil {
		err = fmt.Errorf("skipping sending email to %s. Validation failed: %w", payload.Recipient, err)
		e.logger.Warn(zap.Error(err))
		return nil
	}

	vars := MergeGlobalVars(payload.Variables, *e.globalVars)
	rendered, err := tmplt.Execute(vars)
	if err != nil {
		return err
	}

	email := &mailer.Email{
		Recipients:  []string{payload.Recipient},
		Subject:     rendered.Subject,
		Body:        rendered.Body,
		Attachments: make([]mailer.Attachment, len(payload.Attachments)),
	}
	for i, attachment := range payload.Attachments {
		email.Attachments[i] = mailer.Attachment{
			ContentType: attachment.ContentType,
			Data:        attachment.Data,
			Filename:    attachment.Filename,
		}
	}

	ctx, _ := context.WithTimeout(context.Background(), timeout)
	return e.mailer.Send(ctx, email)
}

func MergeGlobalVars(vars map[string]string, global templates.GlobalVariables) map[string]string {
	if vars == nil {
		vars = make(map[string]string)
	}
	vars["AssetURL"] = global.AssetUrl
	vars["WebURL"] = global.WebAppUrl
	return vars
}
