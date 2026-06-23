package email

import "context"

type SendEmailInput struct {
	To      string
	Subject string
	HTML    string
}

type Provider interface {
	Send(ctx context.Context, input SendEmailInput) error
}
