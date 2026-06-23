package email

import (
	"context"
	"fmt"
	"log"

	"github.com/resend/resend-go/v3"
)

type ResendProvider struct {
	apiKey string
	from   string
}

func NewResendProvider(apiKey, from string) *ResendProvider {
	return &ResendProvider{
		apiKey: apiKey,
		from:   from,
	}
}

func (r *ResendProvider) Send(ctx context.Context, input SendEmailInput) error {
	client := resend.NewClient(r.apiKey)

	params := &resend.SendEmailRequest{
		From:    r.from,
		To:      []string{input.To},
		Subject: input.Subject,
		Html:    input.HTML,
	}
	fmt.Println("params: ", params)

	sent, err := client.Emails.Send(params)
	if err != nil {
		log.Println(err)
		return fmt.Errorf("resend: failed to send email: %w", err)
	}

	log.Printf("Email sent: %s", sent.Id)
	return nil
}
