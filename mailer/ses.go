package mailer

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	"go.uber.org/zap"
	"golang.org/x/net/idna"
)

const (
	SESMailerBackendID = "ses"
	UnknownErrorCode   = "unknown"
)

type SESMailer struct {
	cfg    *SESMailerConfig
	logger *zap.SugaredLogger
	sender string
	svc    *ses.SES
}

// Compile time interface check
var _ Mailer = &SESMailer{}

type SESMailerConfig struct {
	SenderName    string `envconfig:"TIDEPOOL_EMAIL_SENDER_NAME" default:"Tidepool"`
	SenderAddress string `envconfig:"TIDEPOOL_EMAIL_SENDER_ADDRESS" default:"noreply@tidepool.org" validate:"email"`
	Region        string `envconfig:"TIDEPOOL_SES_REGION" default:"us-west-2" validate:"required"`
}

type SESMailerParams struct {
	Cfg    *SESMailerConfig
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
		cfg:    params.Cfg,
		logger: params.Logger.With(zap.String("backend", SESMailerBackendID)),
		sender: FormatSender(params.Cfg.SenderName, params.Cfg.SenderAddress),
		svc:    ses.New(sess),
	}, nil
}

func (s *SESMailer) Send(ctx context.Context, email *Email) error {
	if ctx == nil {
		ctx = context.Background()
	}

	s.logger.Infof("Sending to recipient '%s', from '%s', with CC '%s'", strings.Join(email.Recipients, ", "), s.sender, strings.Join(email.Cc, ", "))

	input, err := CreateSendEmailInput(s.sender, email)
	if err != nil {
		s.logger.Errorw("Error while creating email input", "error", err, "recipients", email.Recipients, "cc", email.Cc)
		return err
	}
	res, err := s.svc.SendEmailWithContext(ctx, input)
	if err != nil {
		code := UnknownErrorCode
		if awsError, ok := err.(awserr.Error); ok {
			code = awsError.Code()
		}

		ObserveError(code, SESMailerBackendID)
		s.logger.Errorw("Error while sending email", "code", code, "error", err)
		return err
	}

	s.logger.Infow("Successfully sent message", "id", *res.MessageId)
	return nil
}

func FormatSender(name, address string) string {
	if name == "" {
		return address
	}
	return fmt.Sprintf("%s <%s>", name, address)
}

func CreateSendEmailInput(sender string, email *Email) (*ses.SendEmailInput, error) {
	toAddresses, err := addresses(email.Recipients)
	if err != nil {
		return nil, err
	}
	ccAddresses, err := addresses(email.Cc)
	if err != nil {
		return nil, err
	}
	return &ses.SendEmailInput{
		Destination: &ses.Destination{
			ToAddresses: toAddresses,
			CcAddresses: ccAddresses,
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

func addresses(emails []string) ([]*string, error) {
	addr := make([]*string, 0, len(emails))
	for _, recipient := range emails {
		escapedRecipient, err := idna.ToASCII(recipient)
		if err != nil {
			return nil, fmt.Errorf("unable to Punycode email: %w", err)
		}
		addr = append(addr, aws.String(escapedRecipient))
	}
	return addr, nil
}
