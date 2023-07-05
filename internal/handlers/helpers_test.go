package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/redhatinsights/mbop/internal/config"
)

// defaultUsersModule holds the default users module set by the configuration.
var defaultUsersModule = config.Get().UsersModule

// cleanup reverts the "users module" to the one that was present before running the tests.
func cleanup() {
	config.Get().UsersModule = defaultUsersModule
}

// TestSendJSONWithStatusCodeContentTypeHeader tests that the helper function returns an "application/json" value for
// the "Content-Type" header.
func TestSendJSONWithStatusCodeContentTypeHeader(t *testing.T) {
	defer cleanup()

	config.Get().UsersModule = "mock"

	testRouter := chi.NewRouter()
	testRouter.Get("/v3/accounts/{orgID}/users", AccountsV3UsersHandler)

	testServer := httptest.NewServer(testRouter)
	defer testServer.Close()

	fullURL := fmt.Sprintf("%s/v3/accounts/12345/users", testServer.URL)
	response, err := http.Get(fullURL) // nolint because the test server's URL is dynamic.
	if err != nil {
		t.Errorf(`unable to send request to the "AccountsV3UsersHandler" endpoint: %s`, err)
	}

	defer response.Body.Close()

	if response.StatusCode != 200 {
		t.Errorf(`unexpected status code received. Want "%d", got "%d"`, 200, response.StatusCode)
	}

	if response.Header.Get("Content-Type") != "application/json" {
		t.Errorf(`unexpected "Content-Type" header received. Want "%s", got "%s"`, "application/json", response.Header.Get("Content-Type"))
	}
}
