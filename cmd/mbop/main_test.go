package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type TestSuite struct {
	suite.Suite
}

func (suite *TestSuite) SetupSuite() {
}

func (suite *TestSuite) TestJWTGet() {
	testData, _ := os.ReadFile("testdata/jwt.json")
	k8sServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/realms/redhat-external/" {
			w.WriteHeader(http.StatusOK)
			w.Write(testData)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer k8sServer.Close()

	os.Setenv("KEYCLOAK_SERVER", k8sServer.URL)
	os.Setenv("KEYCLOAK_VERSION", "17.0.0")

	sut := httptest.NewServer(getMux())
	defer sut.Close()

	resp, err := http.Get(fmt.Sprintf("%s/v1/jwt", sut.URL))

	assert.Nil(suite.T(), err, "error was not nil")
	assert.Equal(suite.T(), 200, resp.StatusCode, "status code not good")
}

func (suite *TestSuite) TearDownSuite() {
}

func TestExampleTestSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}
