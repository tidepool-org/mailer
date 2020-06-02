package mailer

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	"go.uber.org/zap"
)

const (
	SESMailerBackendID = "ses"
)

type SESMailer struct {
	cfg    *SESMailerConfig
	logger *zap.SugaredLogger
	svc    *ses.SES
}

// Compiler time interface check
var _ Mailer = &SESMailer{}

type SESMailerConfig struct {
	Sender string `envconfig:"TIDEPOOL_EMAIL_SENDER",validate:"email"`
	Region string `envconfig:"TIDEPOOL_SES_REGION",validate:"required"`
}

type SESMailerParams struct {
	Cfg *SESMailerConfig
	Logger *zap.SugaredLogger
}

func NewSESMailer(params *SESMailerParams) (*SESMailer, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(params.Cfg.Region)},
	)
	if err != nil {
		return nil, err
	}

	return &SESMailer{
		cfg: params.Cfg,
		logger: params.Logger.With(zap.String("backend", SESMailerBackendID)),
		svc: ses.New(sess),
	}, nil
}

func (s *SESMailer) Send(email Email) error {
	input, err := CreateSendEmailInput(s.cfg.Sender, email)
	if err != nil {
		return err
	}

	res, err := s.svc.SendEmail(input)
	if err != nil {
		code := "UNKNOWN"
		if awsError, ok := err.(awserr.Error); ok {
			code = awsError.Code()
		}

		ObserveError(code, SESMailerBackendID)
		s.logger.Info("Error while sending email", zap.String("code", code), zap.Error(err))
		return err
	}

	s.logger.Info("Successfully sent message", zap.String("id", *res.MessageId))
	return nil
}

func CreateSendEmailInput(sender string, email Email) (*ses.SendEmailInput, error) {
	return &ses.SendEmailInput{
		Destination: &ses.Destination{
			ToAddresses: addresses(email.Recipients),
			CcAddresses: addresses(email.Cc),
		},
		Message: &ses.Message{
			Body: &ses.Body{
				Html: &ses.Content{
					Charset: aws.String(DefaultCharset),
					Data:    aws.String(email.Body),
				},
			},
			Subject: &ses.Content{
				Charset: aws.String(DefaultCharset),
				Data:    aws.String(email.Subject),
			},
		},
		Source: aws.String(sender),
	}, nil
}

func addresses(emails []string) []*string {
	addr := make([]*string, len(emails))
	for i, recipient := range emails {
		addr[i] = aws.String(recipient)
	}
	return addr
}
