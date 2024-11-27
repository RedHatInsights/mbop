package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	l "github.com/redhatinsights/mbop/internal/logger"
	"github.com/redhatinsights/mbop/internal/models"
)

var (
	validSortOrder = []string{"asc", "des"}
	validQueryBy   = []string{"userId", "orgId"} // Originally orgId was "principal" but in FedRAMP cluster we only have orgId
	validAdminOnly = []string{"true", "false"}
)

func sendJSON(w http.ResponseWriter, data any) {
	sendJSONWithStatusCode(w, data, 200)
}

func sendJSONWithStatusCode(w http.ResponseWriter, data any, code int) {
	b, _ := json.Marshal(data)

	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(code)
	_, err := w.Write(b)
	if err != nil {
		l.Log.Error(err, "error writing response")
	}
}

func do500(w http.ResponseWriter, msg string) {
	doError(w, msg, 500)
}

func do400(w http.ResponseWriter, msg string) {
	doError(w, msg, 400)
}

func do404(w http.ResponseWriter, msg string) {
	doError(w, msg, 404)
}

func doError(w http.ResponseWriter, msg string, code int) {
	l.Log.Info("Error during request", "error", msg, "status", code)
	sendJSONWithStatusCode(w, newResponse(msg), code)
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func initV1UserQuery(r *http.Request) (models.UserV1Query, error) {
	q := models.UserV1Query{}

	sortOrder, err := getSortOrder(r)
	if err != nil {
		return q, err
	}

	queryBy, err := getQueryBy(r)
	if err != nil {
		return q, err
	}

	q.SortOrder = sortOrder
	q.QueryBy = queryBy

	return q, nil
}

func initAccountV3UserQuery(r *http.Request) (models.UserV3Query, error) {
	q := models.UserV3Query{}

	sortOrder, err := getSortOrder(r)
	if err != nil {
		return q, err
	}

	adminOnly, err := getAdminOnly(r)
	if err != nil {
		return q, err
	}

	limit, err := getLimit(r)
	if err != nil {
		return q, err
	}

	offset, err := getOffset(r)
	if err != nil {
		return q, err
	}

	q.SortOrder = sortOrder
	q.AdminOnly = adminOnly
	q.Limit = limit
	q.Offset = offset

	return q, nil
}

func usersToV3Response(users []models.User) models.UserV3Responses {
	r := models.UserV3Responses{Responses: []models.UserV3Response{}}

	for _, user := range users {
		r.AddV3Response(models.UserV3Response{
			ID:         user.ID,
			Email:      user.Email,
			Username:   user.Username,
			FirstName:  user.FirstName,
			LastName:   user.LastName,
			IsActive:   user.IsActive,
			IsOrgAdmin: user.IsOrgAdmin,
			IsInternal: user.IsInternal,
			Locale:     user.Locale,
		})
	}

	return r
}

func getUsernamesFromRequestBody(r *http.Request) (models.UserBody, error) {
	var usernames models.UserBody

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return usernames, fmt.Errorf("failed to read request body: %w", err)
	}
	defer r.Body.Close()

	err = json.Unmarshal(body, &usernames)
	if err != nil {
		return usernames, fmt.Errorf("failed to parse request body: %w, request must include 'users': [] ", err)
	}

	return usernames, nil
}

func getUsersByBody(r *http.Request) (models.UsersByBody, error) {
	var usersByBody models.UsersByBody

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return usersByBody, fmt.Errorf("failed to read request body: %w", err)
	}

	defer r.Body.Close()

	err = json.Unmarshal(body, &usersByBody)
	if err != nil {
		return usersByBody, fmt.Errorf("failed to parse request body: %w", err)
	}

	return usersByBody, nil
}

func getOrgIDFromPath(r *http.Request) string {
	return chi.URLParam(r, "orgID")
}

func getSortOrder(r *http.Request) (string, error) {
	if r.URL.Query().Get("sortOrder") == "" || stringInSlice(r.URL.Query().Get("sortOrder"), validSortOrder) {
		if r.URL.Query().Get("sortOrder") == validSortOrder[1] {
			return "desc", nil
		}

		return r.URL.Query().Get("sortOrder"), nil
	}

	return "", fmt.Errorf("sortOrder must be one of '', %s", strings.Join(validSortOrder, ", "))
}

func getQueryBy(r *http.Request) (string, error) {
	if r.URL.Query().Get("queryBy") == "" || stringInSlice(r.URL.Query().Get("queryBy"), validQueryBy) {
		// Translate bop parameters into AMS parameters
		if r.URL.Query().Get("queryBy") == validQueryBy[0] {
			return "id", nil
		}

		if r.URL.Query().Get("queryBy") == validQueryBy[1] {
			return "organizationId", nil
		}
	} else {
		return "", fmt.Errorf("queryBy must be one of %s", strings.Join(validQueryBy, ", "))
	}

	return "", nil
}

func getAdminOnly(r *http.Request) (bool, error) {
	if r.URL.Query().Get("admin_only") == "" || stringInSlice(r.URL.Query().Get("admin_only"), validAdminOnly) {
		if r.URL.Query().Get("admin_only") == validAdminOnly[0] {
			return true, nil
		}

		return false, nil
	}

	return false, fmt.Errorf("admin_only must be one of %s", strings.Join(validSortOrder, ", "))
}

func getLimit(r *http.Request) (int, error) {
	if r.URL.Query().Get("limit") == "" {
		return defaultLimit, nil
	}

	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		return defaultLimit, fmt.Errorf("limit must be of type int")
	}

	return limit, nil
}

func getOffset(r *http.Request) (int, error) {
	if r.URL.Query().Get("offset") == "" {
		return defaultOffset, nil
	}

	offset, err := strconv.Atoi(r.URL.Query().Get("offset"))
	if err != nil {
		return defaultLimit, fmt.Errorf("offset must be of type int")
	}

	return offset, nil
}
