package mailer

import (
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	DefaultCharset = "UTF-8"
)

type Email struct {
	Recipients []string `validate:"min=1,email"`
	Cc         []string `validate:"email"`
	Subject    string   `validate:"required"`
	Body       string   `validate:"required"`
}

type Mailer interface {
	Send(email Email) error
}

func New(id string, logger *zap.SugaredLogger, validate *validator.Validate) (Mailer, error) {
	switch id {
	case SESMailerBackendID:
		backendConfig := &SESMailerConfig{}
		if err := envconfig.Process("", backendConfig); err != nil {
			return nil, err
		}
		if err := validate.Struct(backendConfig); err != nil {
			return nil, err
		}

		params := &SESMailerParams{
			Cfg: backendConfig,
			Logger: logger,
		}
		return NewSESMailer(params)
	default:
		return nil, errors.New(fmt.Sprintf("unknown mailer backend %s", id))
	}
}