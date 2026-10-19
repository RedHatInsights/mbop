package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/redhatinsights/mbop/internal/config"
	l "github.com/redhatinsights/mbop/internal/logger"
	"github.com/redhatinsights/mbop/internal/models"
	"github.com/redhatinsights/mbop/internal/service/mailer"
)

// SendEmails sends the incoming payload's emails through the configured mailer module.
func SendEmails(w http.ResponseWriter, r *http.Request) {
	switch config.Get().MailerModule {
	case awsModule, printModule:
		// create our mailer (using the correct interface)
		sender, err := mailer.NewMailer()
		if err != nil {
			l.Log.Error(err, "error getting mailer")
			do500(w, "error getting mailer: "+err.Error())
			return
		}

		sendEmails(w, r, sender)
	default:
		CatchAll(w, r)
	}
}

// sendEmails reads the unmarshalls the incoming body, and it verifies that it is correct. It determines if the email
// sender was overridden and resolves the specified non-email users through the configured user module. Finally, it
// determines if the default recipient must be grabbed from the config or if, on the other hand, we must set it to what
// the user asked it to be set.
func sendEmails(w http.ResponseWriter, r *http.Request, sender mailer.Emailer) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		do500(w, "failed to read request body: "+err.Error())
		return
	}
	defer r.Body.Close()

	var emails models.Emails
	err = json.Unmarshal(body, &emails)
	if err != nil {
		do400(w, "failed to parse request body: "+err.Error())
		return
	}

	// The user might have wanted to override the sender that goes in the "from" field of the email.
	var fromAddress string
	if emails.EmailSender != "" {
		fromAddress = emails.EmailSender
	} else {
		fromAddress = config.Get().FromEmail
	}

	for _, email := range emails.Emails {
		// creating a copy in order to pass it down into sub-functions
		email := email

		// Lookup the emails for the given usernames unless the client specified otherwise.
		if !emails.SkipUsersResolution {
			err := mailer.LookupEmailsForUsernames(r.Context(), &email)
			if err != nil {
				l.Log.Error(err, "error translating usernames")
				continue
			}
		}

		// Should the user not specify any recipients, we need to determine if we should use the "default
		// recipient" the user might have specified, or the default one that we set up in the configuration.
		if len(email.Recipients) == 0 {
			if emails.DefaultRecipient == "" {
				email.Recipients = []string{config.Get().ToEmail}
			} else {
				email.Recipients = []string{emails.DefaultRecipient}
			}
		}

		err = sender.SendEmail(r.Context(), &email, fromAddress)
		if err != nil {
			l.Log.Error(err, "Error sending email", "email", email)
		}
	}

	sendJSON(w, newResponse("success"))
}
