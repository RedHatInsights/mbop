package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"

	keycloak "github.com/RedHatInsights/simple-kc-client"
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
)

type User struct {
	Username      string `json:"username"`
	ID            int    `json:"id"`
	Email         string `json:"email"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	AccountNumber string `json:"account_number"`
	AddressString string `json:"address_string"`
	IsActive      bool   `json:"is_active"`
	IsOrgAdmin    bool   `json:"is_org_admin"`
	IsInternal    bool   `json:"is_internal"`
	Locale        string `json:"locale"`
	OrgID         string `json:"org_id"`
	DisplayName   string `json:"display_name"`
	Type          string `json:"type"`
	Entitlements  string `json:"entitlements"`
}

type usersByInput struct {
	PrimaryEmail        string `json:"primaryEmail"`
	EmailStartsWith     string `json:"emailStartsWith"`
	PrincipalStartsWith string `json:"principalStartsWith"`
}

type Resp struct {
	User      User   `json:"user"`
	Mechanism string `json:"mechanism"`
}

type AccV2Resp struct {
	Users     []User `json:"users"`
	UserCount int    `json:"userCount"`
}

type Realm struct {
	Realm     string `json:"realm"`
	PublicKey string `json:"public_key"`
}

type V1UserInput struct {
	Users []string `json:"users"`
}

func findUserById(username string) (*User, error) {
	users, err := getUsers()

	if err != nil {
		return nil, err
	}

	for _, user := range users {
		if user.Username == username {
			return &user, nil
		}
	}
	return nil, fmt.Errorf("User is not known")
}

func findUsersBy(accountNo string, orgId string, adminOnly string, status string, limit int, sortOrder string, input *usersByInput, users *V1UserInput) ([]User, error) {
	usersList, err := getUsers()

	if err != nil {
		return nil, err
	}

	out := []User{}
	for _, user := range usersList {
		// When adminOnly is true, parameter “status” is ignored
		if adminOnly == "true" && !user.IsOrgAdmin {
			continue
		} else {
			if status == "disabled" {
				if user.IsActive {
					continue
				}
			} else if status == "enabled" {
				if !user.IsActive {
					continue
				}
			} else if status != "all" {
				if !user.IsActive {
					continue
				}
			}
		}
		if accountNo != "" && user.AccountNumber != accountNo {
			continue
		}
		if orgId != "" && user.OrgID != orgId {
			continue
		}
		if input != nil {
			if input.PrimaryEmail != "" && user.Email != input.PrimaryEmail {
				continue
			}
			if input.EmailStartsWith != "" && !strings.HasPrefix(user.Email, input.EmailStartsWith) {
				continue
			}
			if input.PrincipalStartsWith != "" && !strings.HasPrefix(user.Username, input.PrincipalStartsWith) {
				continue
			}
		}
		if users != nil {
			found := false
			for _, userCheck := range users.Users {
				if strings.EqualFold(userCheck, user.Username) {
					found = true
				}
			}
			if !found {
				continue
			}
		}
		out = append(out, user)

		if limit > 0 && len(out) >= limit {
			break
		}
	}

	if sortOrder == "des" {
		sort.Slice(out, func(i, j int) bool {
			return strings.Compare(out[i].Username, out[j].Username) == 1
		})
	}

	return out, nil
}

func jwtHandler(w http.ResponseWriter, r *http.Request) {
	resp, err := k.GetJWT("redhat-external")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	fmt.Fprintf(w, resp.PublicKey)
}

func getUser(w http.ResponseWriter, r *http.Request) (*User, error) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return &User{}, fmt.Errorf("no auth header found")
	}
	if !strings.Contains(auth, "Basic") {
		return &User{}, fmt.Errorf("auth header is not basic")
	}

	data, err := base64.StdEncoding.DecodeString(auth[6:])

	if err != nil {
		return &User{}, fmt.Errorf("could not split header")
	}
	parts := strings.Split(string(data), ":")

	username := parts[0]
	password := parts[1]

	if err != nil {
		return &User{}, fmt.Errorf("can't create keycloak client: %s", err.Error())
	}

	_, err = k.GetGenericToken("redhat-external", username, password)

	if err != nil {
		return &User{}, fmt.Errorf("couldn't auth user: %s", err.Error())
	}

	userObj, err := findUserById(username)

	if err != nil {
		return &User{}, fmt.Errorf("couldn't find user: %s", err.Error())
	}
	return userObj, nil
}

func authHandler(w http.ResponseWriter, r *http.Request) {

	userObj, err := getUser(w, r)

	if err != nil {
		http.Error(w, fmt.Sprintf("couldn't auth user: %s", err.Error()), http.StatusForbidden)
		return
	}

	respObj := Resp{
		User:      *userObj,
		Mechanism: "Basic",
	}
	str, err := json.Marshal(respObj)
	if err != nil {
		http.Error(w, "could not create response", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(str))
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
}

func usersV1(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	filt := &V1UserInput{}
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "malformed input", http.StatusInternalServerError)
		return
	}
	if string(data) != "" {
		err = json.Unmarshal(data, filt)
		if err != nil {
			http.Error(w, "malformed input", http.StatusInternalServerError)
			return
		}
	}
	adminOnly := r.URL.Query().Get("admin_only")
	status := r.URL.Query().Get("status")
	sortOrder := r.URL.Query().Get("sortOrder")
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		limit = 0
	}
	users, err := findUsersBy("", "", adminOnly, status, limit, sortOrder, nil, filt)

	if err != nil {
		http.Error(w, "could not get response", http.StatusInternalServerError)
		return
	}

	str, err := json.Marshal(users)
	if err != nil {
		http.Error(w, "could not create response", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(str))
}

type usersSpec struct {
	Username   string              `json:"username"`
	Enabled    bool                `json:"enabled"`
	FirstName  string              `json:"firstName"`
	LastName   string              `json:"lastName"`
	Email      string              `json:"email"`
	Attributes map[string][]string `json:"attributes"`
}

func getUsers() (users []User, err error) {
	resp, err := k.Get("/admin/realms/redhat-external/users?max=2000", "", map[string]string{})
	if err != nil {
		fmt.Printf("\n\n%s\n\n", err.Error())
	}

	obj := &[]usersSpec{}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, obj)

	if err != nil {
		return nil, err
	}

	users = []User{}

	for _, user := range *obj {
		IsActiveRaw := user.Attributes["is_active"][0]
		IsActive, _ := strconv.ParseBool(IsActiveRaw)

		IsOrgAdminRaw := user.Attributes["is_org_admin"][0]
		IsOrgAdmin, _ := strconv.ParseBool(IsOrgAdminRaw)

		IsInternalRaw := user.Attributes["is_org_admin"][0]
		IsInternal, _ := strconv.ParseBool(IsInternalRaw)

		IDRaw := user.Attributes["account_id"][0]
		ID, _ := strconv.Atoi(IDRaw)

		OrgID := user.Attributes["org_id"][0]

		var entitle string

		if len(user.Attributes["newEntitlements"]) != 0 {
			entitle = fmt.Sprintf("{%s}", strings.Join(user.Attributes["newEntitlements"], ","))

		} else {
			entitle = user.Attributes["entitlements"][0]
		}

		users = append(users, User{
			Username:      user.Username,
			ID:            ID,
			Email:         user.Email,
			FirstName:     user.FirstName,
			LastName:      user.LastName,
			AccountNumber: user.Attributes["account_number"][0],
			AddressString: "unknown",
			IsActive:      IsActive,
			IsOrgAdmin:    IsOrgAdmin,
			IsInternal:    IsInternal,
			Locale:        "en_US",
			OrgID:         OrgID,
			DisplayName:   user.FirstName,
			Type:          "User",
			Entitlements:  entitle,
		})
	}
	fmt.Printf("%v", obj)
	return users, nil
}

func usersV1Handler(w http.ResponseWriter, r *http.Request) {
	urlParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
	accountId := urlParts[2]
	switch {
	case urlParts[3] == "users" && r.Method == "GET":
		adminOnly := r.URL.Query().Get("admin_only")
		status := r.URL.Query().Get("status")
		limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
		if err != nil {
			limit = 0
		}
		users, err := findUsersBy(accountId, "", adminOnly, status, limit, "", nil, nil)
		if err != nil {
			http.Error(w, "could not get response", http.StatusInternalServerError)
			return
		}

		str, err := json.Marshal(users)
		if err != nil {
			http.Error(w, "could not create response", http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, string(str))
	case urlParts[3] == "usersBy" && r.Method == "POST":
		filt := &usersByInput{}
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "malformed input", http.StatusInternalServerError)
			return
		}
		if string(data) != "" {
			err = json.Unmarshal(data, filt)
			if err != nil {
				http.Error(w, "malformed input", http.StatusInternalServerError)
				return
			}
		}
		adminOnly := r.URL.Query().Get("admin_only")
		status := r.URL.Query().Get("status")
		limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
		if err != nil {
			limit = 0
		}
		users, err := findUsersBy(accountId, "", adminOnly, status, limit, "", filt, nil)
		if err != nil {
			http.Error(w, "could not get response", http.StatusInternalServerError)
			return
		}
		str, err := json.Marshal(users)
		if err != nil {
			http.Error(w, "could not create response", http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, string(str))
	}
}

func usersV2V3Handler(w http.ResponseWriter, r *http.Request) {
	urlParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
	accountId := ""
	orgId := ""
	if urlParts[0] == "v2" {
		accountId = urlParts[2]
	} else {
		orgId = urlParts[2]
	}
	adminOnly := r.URL.Query().Get("admin_only")
	status := r.URL.Query().Get("status")
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		limit = 0
	}

	obj := &usersByInput{}
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}
	if len(data) > 0 {
		err = json.Unmarshal(data, obj)
		if err != nil {
			return
		}
	}

	users, err := findUsersBy(accountId, orgId, adminOnly, status, limit, "", obj, nil)
	if err != nil {
		http.Error(w, "could not get response", http.StatusInternalServerError)
		return
	}
	respObj := AccV2Resp{
		Users:     users,
		UserCount: len(users),
	}
	str, err := json.Marshal(respObj)
	if err != nil {
		http.Error(w, "could not create response", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(str))
}

func entitlements(w http.ResponseWriter, r *http.Request) {
	ALL_PASS := os.Getenv("ALL_PASS")

	if ALL_PASS != "" {
		fmt.Printf("ALL_PASS")
		fmt.Fprint(w, "{\"ansible\": {\"is_entitled\": true, \"is_trial\": false}, \"cost_management\": {\"is_entitled\": true, \"is_trial\": false}, \"insights\": {\"is_entitled\": true, \"is_trial\": false}, \"advisor\": {\"is_entitled\": true, \"is_trial\": false}, \"migrations\": {\"is_entitled\": true, \"is_trial\": false}, \"openshift\": {\"is_entitled\": true, \"is_trial\": false}, \"settings\": {\"is_entitled\": true, \"is_trial\": false}, \"smart_management\": {\"is_entitled\": true, \"is_trial\": false}, \"subscriptions\": {\"is_entitled\": true, \"is_trial\": false}, \"user_preferences\": {\"is_entitled\": true, \"is_trial\": false}, \"notifications\": {\"is_entitled\": true, \"is_trial\": false}, \"integrations\": {\"is_entitled\": true, \"is_trial\": false}, \"automation_analytics\": {\"is_entitled\": true, \"is_trial\": false}}")
		return
	}

	userObj, err := getUser(w, r)

	if err != nil {
		http.Error(w, fmt.Sprintf("couldn't auth user: %s", err.Error()), http.StatusForbidden)
		return
	}

	fmt.Fprint(w, string(userObj.Entitlements))
}

func mainHandler(w http.ResponseWriter, r *http.Request) {
	log.Info(fmt.Sprintf("%s %s %s\n", r.RemoteAddr, r.Method, r.URL))
	switch {
	case r.URL.Path == "/":
		statusHandler(w, r)
	case r.URL.Path == "/v1/users":
		usersV1(w, r)
	case r.URL.Path == "/v1/jwt":
		jwtHandler(w, r)
	case r.URL.Path == "/v1/auth":
		authHandler(w, r)
	case r.URL.Path[:12] == "/v1/accounts":
		usersV1Handler(w, r)
	case r.URL.Path[:12] == "/v2/accounts":
		usersV2V3Handler(w, r)
	case r.URL.Path[:12] == "/v3/accounts":
		usersV2V3Handler(w, r)
	case r.URL.Path == "/api/entitlements/v1/services":
		entitlements(w, r)
	}
}

var k *keycloak.KeyCloakClient
var log logr.Logger

func getMux() *http.ServeMux {

	KEYCLOAK_SERVER := os.Getenv("KEYCLOAK_SERVER")
	KEYCLOAK_USERNAME := os.Getenv("KEYCLOAK_USERNAME")
	KEYCLOAK_PASSWORD := os.Getenv("KEYCLOAK_PASSWORD")
	KEYCLOAK_VERSION := os.Getenv("KEYCLOAK_VERSION")
	if KEYCLOAK_USERNAME == "" {
		KEYCLOAK_USERNAME = "admin"
	}
	if KEYCLOAK_PASSWORD == "" {
		KEYCLOAK_PASSWORD = "admin"
	}
	if KEYCLOAK_VERSION == "" {
		KEYCLOAK_VERSION = "11.0.0"
	}

	zapLog, err := zap.NewDevelopment()
	if err != nil {
		panic(fmt.Sprintf("who watches the watchmen (%v)?", err))
	}
	log = zapr.NewLogger(zapLog)

	key, err := keycloak.NewKeyCloakClient(KEYCLOAK_SERVER, KEYCLOAK_USERNAME, KEYCLOAK_PASSWORD, context.Background(), "master", log, KEYCLOAK_VERSION)

	if err != nil {
		panic(err)
	}

	k = key

	mux := http.NewServeMux()
	mux.HandleFunc("/", mainHandler)
	return mux
}

func main() {
	if err := http.ListenAndServe(":8090", getMux()); err != nil {
		log.Error(err, "reason", "server couldn't start")
	}
}
