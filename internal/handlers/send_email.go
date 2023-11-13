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

func SendEmails(w http.ResponseWriter, r *http.Request) {
	switch config.Get().MailerModule {
	case awsModule, printModule:
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

		// create our mailer (using the correct interface)
		sender, err := mailer.NewMailer()
		if err != nil {
			l.Log.Error(err, "error getting mailer")
			do500(w, "error getting mailer: "+err.Error())
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

				if len(email.Recipients) == 0 {
					email.Recipients = []string{config.Get().ToEmail}
				}
			}

			// Should the user not specify any recipients, we need to determine if we should use the "default
			// recipient" the user might have specified, or the default one that we set up in the configuration.
			if len(email.Recipients) == 0 {
				if emails.DefaultRecipient != "" {
					email.Recipients = []string{emails.DefaultRecipient}
				} else {
					email.Recipients = []string{config.Get().ToEmail}
				}
			}

			err = sender.SendEmail(r.Context(), &email, fromAddress)
			if err != nil {
				l.Log.Error(err, "Error sending email", "email", email)
			}
		}

		sendJSON(w, newResponse("success"))

	default:
		CatchAll(w, r)
	}
}
