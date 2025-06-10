package handlers

import (
	"net/http"

	"github.com/redhatinsights/mbop/internal/config"
	"github.com/redhatinsights/mbop/internal/service/keycloak"
	keycloakuserservice "github.com/redhatinsights/mbop/internal/service/keycloak-user-service"

	"github.com/redhatinsights/mbop/internal/service/ocm"
)

func UsersV1Handler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	switch config.Get().UsersModule {
	case amsModule, mockModule:
		usernames, err := getUsernamesFromRequestBody(r)
		if err != nil {
			do400(w, err.Error())
		}

		q, err := initV1UserQuery(r)
		if err != nil {
			do400(w, err.Error())
			return
		}

		// Create new SDK client
		client, err := ocm.NewOcmClient()
		if err != nil {
			do400(w, err.Error())
			return
		}

		err = client.InitSdkConnection(ctx)
		if err != nil {
			do500(w, "Can't build sdk connection: "+err.Error())
			return
		}

		u, err := client.GetUsers(usernames, q)
		if err != nil {
			do500(w, "Cant Retrieve Accounts: "+err.Error())
			return
		}

		// For each user see if it's an org_admin
		isOrgAdmin, err := client.GetOrgAdmin(u.Users)
		if err != nil {
			do500(w, "Cant Retrieve Role Bindings: "+err.Error())
			return
		}

		for i, user := range u.Users {
			response, ok := isOrgAdmin[user.ID]
			if ok {
				u.Users[i].IsOrgAdmin = response.IsOrgAdmin
			} else {
				user.IsOrgAdmin = false
			}
		}

		// Close SDK Connection
		client.CloseSdkConnection()

		sendJSON(w, u.Users)
	case keycloakModule:
		usernames, err := getUsernamesFromRequestBody(r)
		if err != nil {
			do400(w, err.Error())
		}

		q, err := initV1UserQuery(r)
		if err != nil {
			do400(w, err.Error())
			return
		}

		keycloakClient := keycloak.NewKeyCloakClient()
		err = keycloakClient.InitKeycloakConnection()
		if err != nil {
			do500(w, "Can't build keycloak connection: "+err.Error())
			return
		}

		token, err := keycloakClient.GetAccessToken()
		if err != nil {
			do500(w, "Can't fetch keycloak token: "+err.Error())
			return
		}

		userServiceClient, err := keycloakuserservice.NewKeyCloakUserServiceClient()
		if err != nil {
			do500(w, "Can't build keycloak user service client: "+err.Error())
			return
		}

		err = userServiceClient.InitKeycloakUserServiceConnection()
		if err != nil {
			do500(w, "Can't build keycloak user service connection: "+err.Error())
			return
		}

		u, err := userServiceClient.GetUsers(token, usernames, q)
		if err != nil {
			do500(w, "Cant Retrieve Keycloak Accounts: "+err.Error())
			return
		}

		if u.UserCount == 1 {
			sendJSON(w, u.Users)
			return
		}

		sendJSON(w, u)
		return
	default:
		// mbop server instance injected somewhere
		// pass right through to the current handler
		CatchAll(w, r)
	}
}
