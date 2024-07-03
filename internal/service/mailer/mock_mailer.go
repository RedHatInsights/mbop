package mailer

import (
	"context"

	"github.com/redhatinsights/mbop/internal/models"
)

// MockEmailer is a mocked mailer that is intended to be used for tests.
type MockEmailer struct {
	Emails      []*models.Email
	FromAddress string
}

// Build flag.
var _ = (Emailer)(&printEmailer{})

// SendEmail copies the incoming arguments into the struct to be able to check them afterwards in the tests.
func (m *MockEmailer) SendEmail(_ context.Context, email *models.Email, fromAddress string) error {
	m.Emails = append(m.Emails, email)
	m.FromAddress = fromAddress

	return nil
}
