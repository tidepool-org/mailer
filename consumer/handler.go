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
	globalVars *templates.GlobalVariables
	logger     *zap.SugaredLogger
	mailer     mailer.Mailer
	tmplts     templates.Templates
}

var _ events.EmailEventHandler = &EmailEventHandler{}

func NewEmailEventHandler(logger *zap.SugaredLogger, mailer mailer.Mailer, tmplts templates.Templates, globalVars *templates.GlobalVariables) (*EmailEventHandler, error) {
	return &EmailEventHandler{
		globalVars: globalVars,
		logger:     logger,
		mailer:     mailer,
		tmplts:     tmplts,
	}, nil
}

func (e *EmailEventHandler) HandleSendEmailTemplate(payload events.SendEmailTemplateEvent) error {
	tmplt, ok := e.tmplts[templates.TemplateName(payload.Template)]
	if !ok {
		e.logger.Info("Skipping email to %s because template %s doesn't exist", payload.Recipient, payload.Template)
		return nil
	}

	vars := MergeGlobalVars(payload.Variables, *e.globalVars)
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

func MergeGlobalVars(vars map[string]string, global templates.GlobalVariables) map[string]string {
	if vars == nil {
		vars = make(map[string]string)
	}
	vars["AssetURL"] = global.AssetUrl
	vars["WebURL"] = global.WebAppUrl
	return vars
}
