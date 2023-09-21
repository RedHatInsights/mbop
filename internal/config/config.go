package config

import (
	"os"
	"strconv"
)

type MbopConfig struct {
	FromEmail              string
	ToEmail                string
	SESRegion              string
	SESAccessKey           string
	SESSecretKey           string
	MailerModule           string
	JwtModule              string
	JwkURL                 string
	UsersModule            string
	CognitoAppClientID     string
	CognitoAppClientSecret string
	CognitoScope           string
	OauthTokenURL          string
	AmsURL                 string
	TokenTTL               string
	TokenKID               string
	PrivateKey             string
	PublicKey              string
	DisableCatchall        bool
	IsInternalLabel        string
	Debug                  bool

	KeyCloakUserServiceScheme  string
	KeyCloakUserServiceHost    string
	KeyCloakUserServicePort    string
	KeyCloakUserServiceTimeout int64
	KeyCloakTimeout            int64
	KeyCloakTokenURL           string
	KeyCloakTokenPath          string
	KeyCloakTokenUsername      string
	KeyCloakTokenPassword      string
	KeyCloakTokenGrantType     string
	KeyCloakTokenClientID      string

	StoreBackend     string
	DatabaseHost     string
	DatabasePort     string
	DatabaseUser     string
	DatabasePassword string
	DatabaseName     string

	Port    string
	TLSPort string
	UseTLS  bool
	CertDir string
}

var conf *MbopConfig

func Get() *MbopConfig {
	if conf != nil {
		return conf
	}

	disableCatchAll, _ := strconv.ParseBool(fetchWithDefault("DISABLE_CATCHALL", "false"))
	debug, _ := strconv.ParseBool(fetchWithDefault("DEBUG", "false"))
	certDir := fetchWithDefault("CERT_DIR", "/certs")
	keyCloakTimeout, _ := strconv.ParseInt(fetchWithDefault("KEYCLOAK_TIMEOUT", "60"), 0, 64)
	userServiceTimeout, _ := strconv.ParseInt(fetchWithDefault("KEYCLOAK_USER_SERVICE_TIMEOUT", "60"), 0, 64)

	var tls bool
	_, err := os.Stat(certDir + "/tls.crt")
	if err == nil {
		tls = true
	}

	c := &MbopConfig{
		UsersModule:     fetchWithDefault("USERS_MODULE", ""),
		JwtModule:       fetchWithDefault("JWT_MODULE", ""),
		JwkURL:          fetchWithDefault("JWK_URL", ""),
		MailerModule:    fetchWithDefault("MAILER_MODULE", "print"),
		FromEmail:       fetchWithDefault("FROM_EMAIL", "no-reply@redhat.com"),
		ToEmail:         fetchWithDefault("TO_EMAIL", "no-reply@redhat.com"),
		SESRegion:       fetchWithDefault("SES_REGION", "us-east-1"),
		SESAccessKey:    fetchWithDefault("SES_ACCESS_KEY", ""),
		SESSecretKey:    fetchWithDefault("SES_SECRET_KEY", ""),
		DisableCatchall: disableCatchAll,

		DatabaseHost:     fetchWithDefault("DATABASE_HOST", "localhost"),
		DatabasePort:     fetchWithDefault("DATABASE_PORT", "5432"),
		DatabaseUser:     fetchWithDefault("DATABASE_USER", "postgres"),
		DatabasePassword: fetchWithDefault("DATABASE_PASSWORD", ""),
		DatabaseName:     fetchWithDefault("DATABASE_NAME", "mbop"),
		StoreBackend:     fetchWithDefault("STORE_BACKEND", "memory"),

		CognitoAppClientID:     fetchWithDefault("COGNITO_APP_CLIENT_ID", ""),
		CognitoAppClientSecret: fetchWithDefault("COGNITO_APP_CLIENT_SECRET", ""),
		CognitoScope:           fetchWithDefault("COGNITO_SCOPE", ""),
		OauthTokenURL:          fetchWithDefault("OAUTH_TOKEN_URL", ""),
		AmsURL:                 fetchWithDefault("AMS_URL", ""),
		TokenTTL:               fetchWithDefault("TOKEN_TTL_DURATION", "5m"),
		TokenKID:               fetchWithDefault("TOKEN_KID", ""),
		PrivateKey:             fetchWithDefault("TOKEN_PRIVATE_KEY", ""),
		PublicKey:              fetchWithDefault("TOKEN_PUBLIC_KEY", ""),
		IsInternalLabel:        fetchWithDefault("IS_INTERNAL_LABEL", ""),
		Debug:                  debug,

		KeyCloakUserServiceHost:    fetchWithDefault("KEYCLOAK_USER_SERVICE_HOST", "localhost"),
		KeyCloakUserServicePort:    fetchWithDefault("KEYCLOAK_USER_SERVICE_PORT", ":8000"),
		KeyCloakUserServiceScheme:  fetchWithDefault("KEYCLOAK_USER_SERVICE_SCHEME", "http"),
		KeyCloakUserServiceTimeout: userServiceTimeout,
		KeyCloakTimeout:            keyCloakTimeout,
		KeyCloakTokenURL:           fetchWithDefault("KEYCLOAK_TOKEN_URL", "http://localhost:8080/"),
		KeyCloakTokenPath:          fetchWithDefault("KEYCLOAK_TOKEN_PATH", "realms/master/protocol/openid-connect/token"),
		KeyCloakTokenUsername:      fetchWithDefault("KEYCLOAK_TOKEN_USERNAME", "admin"),
		KeyCloakTokenPassword:      fetchWithDefault("KEYCLOAK_TOKEN_PASSWORD", "admin"),
		KeyCloakTokenGrantType:     fetchWithDefault("KEYCLOAK_TOKEN_GRANT_TYPE", "password"),
		KeyCloakTokenClientID:      fetchWithDefault("KEYCLOAK_TOKEN_CLIENT_ID", "admin-cli"),

		Port:    fetchWithDefault("PORT", "8090"),
		TLSPort: fetchWithDefault("TLS_PORT", "8890"),
		UseTLS:  tls,
		CertDir: certDir,
	}

	conf = c
	return conf
}

func fetchWithDefault(name, defaultValue string) string {
	if v, ok := os.LookupEnv(name); ok {
		return v
	}

	return defaultValue
}

// TO BE USED FROM TESTING ONLY.
func Reset() {
	conf = nil
}
