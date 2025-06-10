package handlers

import (
	"net/http"

	"github.com/redhatinsights/mbop/internal/config"
	"github.com/redhatinsights/mbop/internal/models"
	"github.com/redhatinsights/mbop/internal/service/keycloak"
	keycloakuserservice "github.com/redhatinsights/mbop/internal/service/keycloak-user-service"
	"github.com/redhatinsights/mbop/internal/service/ocm"
)

func AccountsV3UsersByHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	switch config.Get().UsersModule {
	case amsModule, mockModule:
		orgID := getOrgIDFromPath(r)
		if orgID == "" {
			do400(w, "Request URL must include orgID: /v3/accounts/{orgID}/usersBy")
			return
		}

		usersByBody, err := getUsersByBody(r)
		if err != nil {
			do400(w, err.Error())
			return
		}

		if usersByBody == (models.UsersByBody{}) {
			do400(w, "request must include 'primaryEmail', 'emailStartsWith', or 'principalStartsWith'")
			return
		}

		q, err := initAccountV3UserQuery(r)
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
			do500(w, "Can't build connection: "+err.Error())
			return
		}

		u, err := client.GetAccountV3UsersBy(orgID, q, usersByBody)
		if err != nil {
			do500(w, "Cant Retrieve Users: "+err.Error())
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
				if q.AdminOnly { // if admin_only return only the org_admins
					u.RemoveUser(i)
				}
			}
		}

		r := usersToV3Response(u.Users)

		client.CloseSdkConnection()

		sendJSON(w, r.Responses)
	case keycloakModule:
		orgID := getOrgIDFromPath(r)
		if orgID == "" {
			do400(w, "Request URL must include orgID: /v3/accounts/{orgID}/usersBy")
			return
		}

		usersByBody, err := getUsersByBody(r)
		if err != nil {
			do400(w, err.Error())
			return
		}

		if usersByBody == (models.UsersByBody{}) {
			do400(w, "request must include 'primaryEmail', 'emailStartsWith', or 'principalStartsWith'")
			return
		}

		q, err := initAccountV3UserQuery(r)
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

		u, err := userServiceClient.GetAccountV3UsersBy(orgID, token, q, usersByBody)
		if err != nil {
			do500(w, "Cant Retrieve Keycloak Accounts: "+err.Error())
			return
		}

		if q.AdminOnly {
			for i, user := range u.Users {
				if !user.IsOrgAdmin {
					u.RemoveUser(i)
				}
			}
		}

		if len(u.Users) == 1 {
			sendJSON(w, u.Users[0])
			return
		}

		sendJSON(w, u)
	default:
		// mbop server instance injected somewhere
		// pass right through to the current handler
		CatchAll(w, r)
	}
}
