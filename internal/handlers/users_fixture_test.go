package handlers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/redhatinsights/mbop/internal/config"
)

func TestFixtureUsers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "users.json")
	if err := os.WriteFile(path, []byte(`[{
      "lookup_id":"sso-test-subject","id":52599,"org_id":"52599",
      "username":"cpt-user","is_active":true,"is_org_admin":false
    }]`), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("USERS_MODULE", "fixture")
	t.Setenv("USERS_FIXTURE_FILE", path)
	config.Reset()
	t.Cleanup(config.Reset)
	for _, tc := range []struct {
		name, body, query string
		status            int
		contains          string
	}{
		{"known", `{"users":["sso-test-subject"]}`, "userId", 200, `"id":52599`},
		{"repeat", `{"users":["sso-test-subject"]}`, "userId", 200, `"id":52599`},
		{"unknown", `{"users":["missing"]}`, "userId", 200, `[]`},
		{"empty", `{"users":[]}`, "userId", 400, "at least one"},
		{"malformed", `{`, "userId", 400, "at least one"},
		{"unsupported", `{"users":["sso-test-subject"]}`, "orgId", 400, "userId only"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/v1/users?queryBy="+tc.query, strings.NewReader(tc.body))
			UsersV1Handler(w, r)
			if w.Code != tc.status || !strings.Contains(w.Body.String(), tc.contains) {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
		})
	}
	for _, content := range []string{"not-json", "missing-file"} {
		if content == "missing-file" {
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
		} else {
			if err := os.WriteFile(path, []byte(content), 0600); err != nil {
				t.Fatal(err)
			}
		}
		w := httptest.NewRecorder()
		UsersV1Handler(w, httptest.NewRequest(http.MethodPost, "/v1/users?queryBy=userId", strings.NewReader(`{"users":["sso-test-subject"]}`)))
		if w.Code != 500 {
			t.Fatalf("expected fail-closed 500, got %d", w.Code)
		}
	}
}
