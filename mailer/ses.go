package mailer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	"go.uber.org/zap"
	"golang.org/x/net/idna"

	"gopkg.in/gomail.v2"
)

const (
	SESMailerBackendID = "ses"
	UnknownErrorCode   = "unknown"
)

type SESMailer struct {
	cfg    *SESMailerConfig
	logger *zap.SugaredLogger
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
		svc:    ses.New(sess),
	}, nil
}

func (s *SESMailer) Send(ctx context.Context, email *Email) error {
	if ctx == nil {
		ctx = context.Background()
	}

	s.logger.Infof("Sending to recipient '%s', with CC '%s'", strings.Join(email.Recipients, ", "), strings.Join(email.Cc, ", "))

	input, err := s.CreateSendEmailInput(email)
	if err != nil {
		s.logger.Errorw("Error while creating email input", "error", err, "recipients", email.Recipients, "cc", email.Cc)
		return err
	}
	res, err := s.svc.SendRawEmailWithContext(ctx, input)
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

func (s *SESMailer) CreateSendEmailInput(email *Email) (*ses.SendRawEmailInput, error) {
	toAddresses, err := addresses(email.Recipients)
	if err != nil {
		return nil, err
	}
	ccAddresses, err := addresses(email.Cc)
	if err != nil {
		return nil, err
	}

	msg := gomail.NewMessage()

	var recipients []*string
	for _, r := range toAddresses {
		recipients = append(recipients, &r)
	}
	for _, r := range ccAddresses {
		recipients = append(recipients, &r)
	}

	msg.SetHeader("To", email.Recipients...)
	msg.SetAddressHeader("From", s.cfg.SenderAddress, s.cfg.SenderName)
	msg.SetHeader("Subject", email.Subject)
	msg.SetBody("text/html", email.Body)
	if len(email.Cc) > 0 {
		msg.SetHeader("cc", ccAddresses...)
	}

	for _, attachment := range email.Attachments {
		msg.Attach(attachment.Filename, gomail.SetCopyFunc(func(writer io.Writer) error {
			_, err := writer.Write([]byte(attachment.Data))
			return err
		}), gomail.SetHeader(map[string][]string{
			"content-type": {attachment.ContentType},
		}))
	}

	// create a new buffer to add raw data
	var emailRaw bytes.Buffer
	if _, err = msg.WriteTo(&emailRaw); err != nil {
		return nil, err
	}

	message := ses.RawMessage{Data: emailRaw.Bytes()}

	return &ses.SendRawEmailInput{
		Source:       &s.cfg.SenderAddress,
		Destinations: recipients,
		RawMessage:   &message,
	}, nil
}

func addresses(emails []string) ([]string, error) {
	addr := make([]string, 0, len(emails))
	for _, recipient := range emails {
		escapedRecipient, err := idna.ToASCII(recipient)
		if err != nil {
			return nil, fmt.Errorf("unable to Punycode email: %w", err)
		}
		addr = append(addr, escapedRecipient)
	}
	return addr, nil
}
