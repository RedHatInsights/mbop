package mailer

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/redhatinsights/mbop/internal/config"
	"github.com/redhatinsights/mbop/internal/models"
)

type Emailer interface {
	SendEmail(ctx context.Context, email *models.Email, fromAddress string) error
}

func NewMailer() (Emailer, error) {
	var sender Emailer

	switch config.Get().MailerModule {
	case "aws":
		if cfg == nil {
			return nil, errors.New("aws config not initialized")
		}

		sender = &awsSESEmailer{client: sesv2.NewFromConfig(*cfg)}
	case "mock":
		sender = &MockEmailer{}
	case "print":
		sender = &printEmailer{}
	default:
		return nil, fmt.Errorf("unsupported mailer module %q", config.Get().MailerModule)
	}

	return sender, nil
}
