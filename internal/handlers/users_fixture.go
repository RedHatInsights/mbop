package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"

	"github.com/redhatinsights/mbop/internal/config"
	"github.com/redhatinsights/mbop/internal/models"
)

// fixtureUser separates an SSO subject (often a UUID) from BOP's numeric ID.
type fixtureUser struct {
	LookupID string `json:"lookup_id"`
	models.User
}

// fixtureUsers is opt-in and never falls back to a real identity service.
func fixtureUsers(w http.ResponseWriter, r *http.Request) {
	body, err := getUsernamesFromRequestBody(r)
	if err != nil || len(body.Users) == 0 {
		do400(w, "request must include at least one user")
		return
	}
	query, err := initV1UserQuery(r)
	if err != nil {
		do400(w, err.Error())
		return
	}
	if query.QueryBy != "id" {
		do400(w, "fixture backend supports queryBy=userId only")
		return
	}
	raw, err := os.ReadFile(config.Get().UsersFixtureFile)
	if err != nil {
		do500(w, "cannot read user fixtures")
		return
	}
	var fixtures []fixtureUser
	if err := json.Unmarshal(raw, &fixtures); err != nil {
		do500(w, "cannot decode user fixtures")
		return
	}
	users := []models.User{}
	for _, user := range fixtures {
		lookup := user.LookupID
		if lookup == "" {
			lookup = strconv.Itoa(user.ID)
		}
		for _, requested := range body.Users {
			if lookup == requested {
				users = append(users, user.User)
				break
			}
		}
	}
	sendJSON(w, users)
}
