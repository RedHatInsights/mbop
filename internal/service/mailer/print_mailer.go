package mailer

import (
	"context"
	"fmt"

	"github.com/redhatinsights/mbop/internal/models"
)

// this is the default mailer - it just prints the message to stdout (not logging it)
type printEmailer struct{}

var _ = (Emailer)(&printEmailer{})

func (p printEmailer) SendEmail(_ context.Context, email *models.Email, fromAddress string) error {
	l := 50
	if len(email.Body) < 50 {
		l = len(email.Body)
	}

	fmt.Printf(`From: %v
To: %v
CC: %v
BCC: %v
Subject: %v
BodyType: %v
Message: %v...(truncated to 50 chars)
`, fromAddress, email.Recipients, email.CcList, email.BccList, email.Subject, email.BodyType, email.Body[:l])

	return nil
}
