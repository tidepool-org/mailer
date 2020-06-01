package mailer

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
