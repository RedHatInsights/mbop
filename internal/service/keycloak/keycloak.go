package keycloak

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/redhatinsights/mbop/internal/config"
	"github.com/redhatinsights/mbop/internal/models"
)

type Client struct {
	client *http.Client
}

func (keycloak *Client) InitKeycloakConnection() error {
	keycloak.client = &http.Client{
		Timeout: time.Duration(config.Get().KeyCloakTimeout * int64(time.Second)),
	}

	return nil
}

func (keycloak *Client) GetAccessToken() (string, error) {
	token := models.KeycloakTokenObject{}
	url, err := createTokenURL()
	if err != nil {
		return "", err
	}

	body := createEncodedTokenBody()

	resp, err := http.Post(url.String(), "application/x-www-form-urlencoded", body)
	if err != nil {
		return "", fmt.Errorf("error fetching keycloak token response: %s", err)
	}

	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading keycloak token response body: %s", err)
	}

	err = json.Unmarshal(responseBody, &token)
	if err != nil {
		return "", fmt.Errorf("error unmarshling keycloak token response: %s", err)
	}

	return token.AccessToken, nil
}

func createEncodedTokenBody() *strings.Reader {
	data := url.Values{}
	data.Set("username", config.Get().KeyCloakTokenUsername)
	data.Set("grant_type", config.Get().KeyCloakTokenGrantType)
	data.Set("client_id", config.Get().KeyCloakTokenClientID)

	if config.Get().KeyCloakTokenGrantType == "password" {
		data.Set("password", config.Get().KeyCloakTokenPassword)
	} else {
		data.Set("client_secret", config.Get().KeyCloakTokenPassword)
	}

	return strings.NewReader(data.Encode())
}

func createTokenURL() (*url.URL, error) {
	url, err := url.Parse(fmt.Sprintf("%s%s", config.Get().KeyCloakTokenURL, config.Get().KeyCloakTokenPath))
	if err != nil {
		return nil, fmt.Errorf("error creating keycloak token url: %s", err)
	}

	return url, err
}
