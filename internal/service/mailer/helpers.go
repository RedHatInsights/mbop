package mailer

import (
	"context"
	"fmt"
	"strings"

	"github.com/redhatinsights/mbop/internal/config"
	l "github.com/redhatinsights/mbop/internal/logger"
	"github.com/redhatinsights/mbop/internal/models"
	"github.com/redhatinsights/mbop/internal/service/keycloak"
	keycloakuserservice "github.com/redhatinsights/mbop/internal/service/keycloak-user-service"
	"github.com/redhatinsights/mbop/internal/service/ocm"
	"golang.org/x/exp/maps"
)

func LookupEmailsForUsernames(ctx context.Context, email *models.Email) error {
	// Generate a map of names to look up from the email to/cc/bcc
	toLookup := make(map[string]string)
	for _, name := range email.Recipients {
		if !strings.Contains(name, "@") {
			toLookup[name] = ""
		}
	}
	for _, name := range email.CcList {
		if !strings.Contains(name, "@") {
			toLookup[name] = ""
		}
	}
	for _, name := range email.BccList {
		if !strings.Contains(name, "@") {
			toLookup[name] = ""
		}
	}

	// nothing to lookup
	if len(toLookup) == 0 {
		return nil
	}

	l.Log.Info("Looking up usernames", "user_module", config.Get().UsersModule, "usernames", maps.Keys(toLookup))

	// using the appropriate AMS Module - search and look up the emails from the
	// usernames
	switch config.Get().UsersModule {
	case "ams":
		ocm, err := ocm.NewOcmClient()
		if err != nil {
			return err
		}

		err = ocm.InitSdkConnection(ctx)
		if err != nil {
			return err
		}

		users, err := ocm.GetUsers(models.UserBody{Users: maps.Keys(toLookup)}, models.UserV1Query{})
		if err != nil {
			return err
		}

		for _, user := range users.Users {
			toLookup[user.Username] = user.Email
		}
	case "keycloak":
		keycloakClient := keycloak.NewKeyCloakClient()
		err := keycloakClient.InitKeycloakConnection()
		if err != nil {
			return err
		}

		token, err := keycloakClient.GetAccessToken()
		if err != nil {
			return err
		}

		userServiceClient, err := keycloakuserservice.NewKeyCloakUserServiceClient()
		if err != nil {
			return err
		}

		err = userServiceClient.InitKeycloakUserServiceConnection()
		if err != nil {
			return err
		}

		u, err := userServiceClient.GetUsers(token, models.UserBody{Users: maps.Keys(toLookup)}, models.UserV1Query{})
		if err != nil {
			return err
		}

		for _, user := range u.Users {
			toLookup[user.Username] = user.Email
		}
	case "mock":
		for k := range toLookup {
			toLookup[k] = k + "@mocked.biz"
		}
	default:
		return fmt.Errorf("no configured user module for username translations")
	}

	// ...and finally, replace the usernames -> in the lists on the email objects
	for i, name := range email.Recipients {
		if emailAddr, ok := toLookup[name]; ok {
			email.Recipients[i] = emailAddr
		}
	}
	for i, name := range email.CcList {
		if emailAddr, ok := toLookup[name]; ok {
			email.CcList[i] = emailAddr
		}
	}
	for i, name := range email.BccList {
		if emailAddr, ok := toLookup[name]; ok {
			email.BccList[i] = emailAddr
		}
	}

	return nil
}
