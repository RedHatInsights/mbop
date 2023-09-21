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

		for _, email := range emails.Emails {
			// creating a copy in order to pass it down into sub-functions
			email := email

			err := mailer.LookupEmailsForUsernames(r.Context(), &email)
			if err != nil {
				l.Log.Error(err, "error translating usernames")
				continue
			}

			if len(email.Recipients) == 0 {
				email.Recipients = []string{config.Get().ToEmail}
			}

			err = sender.SendEmail(r.Context(), &email)
			if err != nil {
				l.Log.Error(err, "Error sending email", "email", email)
			}
		}

		sendJSON(w, newResponse("success"))

	default:
		CatchAll(w, r)
	}
}
