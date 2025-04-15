package handlers

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/redhatinsights/mbop/internal/config"
	l "github.com/redhatinsights/mbop/internal/logger"
	"github.com/redhatinsights/mbop/internal/models"
	"github.com/redhatinsights/mbop/internal/service/mailer"
)

// TestMain initializes the logger before running the tests so that we don't suffer from "panics" when the code under
// test attempts to log information.
func TestMain(m *testing.M) {
	err := l.Init()
	if err != nil {
		log.Fatalln(err)
	}

	exitCode := m.Run()

	os.Exit(exitCode)
}

// slicesAreEqual returns true if the given slices are equal.
func slicesAreEqual(t *testing.T, s1 []string, s2 []string) bool {
	if len(s1) != len(s2) {
		t.Errorf(`slices are not of the same length. Want "%s" and "%s" to be equal`, s1, s2)

		return false
	}

	for i := 0; i < len(s1); i++ {
		if s1[i] != s2[i] {
			t.Errorf(`slices do not contain the same elements. Want "%s" and "%s" to be equal`, s1, s2)

			return false
		}
	}

	return true
}

// TestSendEmailsFailedParseBody tests that an invalid incoming payload results in a "bad request" response.
func TestSendEmailsFailedParseBody(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/v1/sendEmails", nil)
	writer := httptest.NewRecorder()

	// Call the function under test.
	SendEmails(writer, request)

	response := writer.Result()
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadRequest {
		t.Errorf(`want "%d" status code when sending an invalid JSON payload, got "%d"`, http.StatusBadRequest, response.StatusCode)
	}
}

// TestSendEmails tests that the incoming emails are correctly parsed and processed, and that the output emails to be
// sent to the mailers contain the incoming email's data.
func TestSendEmails(t *testing.T) {
	// Mock the incoming payload.
	recipients := []string{"a@redhat.com", "a"}
	ccList := []string{"copy@redhat.com", "b"}
	bccList := []string{"hiddenCopy@redhat.com", "c"}

	email := &models.Email{
		Subject:    "subject",
		Body:       "body",
		Recipients: recipients,
		CcList:     ccList,
		BccList:    bccList,
		BodyType:   "html",
	}

	emails := &models.Emails{
		Emails: []models.Email{*email},
	}

	// Marshal the mocked body.
	requestBody, err := json.Marshal(emails)
	if err != nil {
		t.Fatalf("unable to marshal the emails model to JSON: %s", err)
	}

	// Mock the request and the response writer.
	request := httptest.NewRequest(http.MethodPost, "/v1/sendEmails", bytes.NewBuffer(requestBody))
	writer := httptest.NewRecorder()

	// Change the mailer module to the "mock" one, and make sure that it gets reverted after the test.
	originalMailerModule := config.Get().MailerModule
	defer func() {
		config.Get().MailerModule = originalMailerModule
	}()
	config.Get().MailerModule = "mock"

	// Use the "mock emailer" module for the test.
	sender := &mailer.MockEmailer{}

	// Change the users' module to "mock", and then rever the value to the original one after the test.
	originalUsersModule := config.Get().UsersModule
	defer func() {
		config.Get().UsersModule = originalUsersModule
	}()
	config.Get().UsersModule = "mock"

	// Call the function under test.
	sendEmails(writer, request, sender)

	// Assert that the operation succeeded.
	response := writer.Result()
	defer response.Body.Close()

	// Assert that we are returning the expected status code.
	if response.StatusCode != http.StatusOK {
		t.Errorf(`want "%d" status code when the email sending operation succeeds, got "%d"`, http.StatusOK, response.StatusCode)
	}

	// Assert that the returned body is the correct one.
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("unable to read the response body after sending an emai: %s", err)
	}

	expectedMessageBody := `{"message":"success"}`
	if string(responseBody) != expectedMessageBody {
		t.Errorf(`unexpected response body. Want "%s", got "%s"`, expectedMessageBody, string(responseBody))
	}

	// Assert that we did not accidentally modify the "from address".
	if config.Get().FromEmail != sender.FromAddress {
		t.Errorf(`the "from address" email should not have been modified by the test. Want "%s", got "%s"`, config.Get().FromEmail, sender.FromAddress)
	}

	// Assert that only one email "was sent".
	if 1 != len(sender.Emails) {
		t.Errorf("want 1 email to be sent, got %d sent", len(sender.Emails))
	}

	// Assert that the sent email contains the expected fields.
	sentEmail := sender.Emails[0]

	// Assert that the subject is correct.
	if email.Subject != sentEmail.Subject {
		t.Errorf(`unexpected email subject sent. Want "%s", got "%s"`, email.Subject, sentEmail.Subject)
	}

	// Assert that the body is correct.
	if email.Body != sentEmail.Body {
		t.Errorf(`unexpected email body sent. Want "%s", got "%s"`, email.Body, sentEmail.Body)
	}

	// Assert that the recipients are correct.
	email.Recipients[1] = "a@mocked.biz"
	if !slicesAreEqual(t, email.Recipients, sentEmail.Recipients) {
		t.Error(`unexpected recipients specified in the sent email`)
	}

	// Assert that the CC recipients are correct.
	email.CcList[1] = "b@mocked.biz"
	if !slicesAreEqual(t, email.CcList, sentEmail.CcList) {
		t.Error(`unexpected CC recipients specified in the sent email`)
	}

	// Assert that the BCC recipients are correct.
	email.BccList[1] = "c@mocked.biz"
	if !slicesAreEqual(t, email.BccList, sentEmail.BccList) {
		t.Error(`unexpected BCC recipients specified in the sent email`)
	}

	// Assert that the specified body type is correct.
	if email.BodyType != sentEmail.BodyType {
		t.Errorf(`unexpected email body type sent. Want "%s", got "%s"`, email.BodyType, sentEmail.BodyType)
	}
}

// TestSendEmailsOverrideFromAddress tests that the incoming emails are correctly parsed and processed, and that the
// output emails to be sent to the mailers contain the incoming email's data. The "from" address is overridden in the
// incoming payload.
func TestSendEmailsOverrideFromAddress(t *testing.T) {
	// Mock the incoming payload.
	recipients := []string{"a@redhat.com", "a"}
	ccList := []string{"copy@redhat.com", "b"}
	bccList := []string{"hiddenCopy@redhat.com", "c"}

	email := &models.Email{
		Subject:    "subject",
		Body:       "body",
		Recipients: recipients,
		CcList:     ccList,
		BccList:    bccList,
		BodyType:   "html",
	}

	overriddenFromAddress := "custom-from-address@redhat.com"
	emails := &models.Emails{
		Emails:      []models.Email{*email},
		EmailSender: overriddenFromAddress,
	}

	// Marshal the mocked body.
	requestBody, err := json.Marshal(emails)
	if err != nil {
		t.Fatalf("unable to marshal the emails model to JSON: %s", err)
	}

	// Mock the request and the response writer.
	request := httptest.NewRequest(http.MethodPost, "/v1/sendEmails", bytes.NewBuffer(requestBody))
	writer := httptest.NewRecorder()

	// Change the mailer module to the "mock" one, and make sure that it gets reverted after the test.
	originalMailerModule := config.Get().MailerModule
	defer func() {
		config.Get().MailerModule = originalMailerModule
	}()
	config.Get().MailerModule = "mock"

	// Use the "mock emailer" module for the test.
	sender := &mailer.MockEmailer{}

	// Change the users' module to "mock", and then rever the value to the original one after the test.
	originalUsersModule := config.Get().UsersModule
	defer func() {
		config.Get().UsersModule = originalUsersModule
	}()
	config.Get().UsersModule = "mock"

	// Call the function under test.
	sendEmails(writer, request, sender)

	// Assert that the operation succeeded.
	response := writer.Result()
	defer response.Body.Close()

	// Assert that we are returning the expected status code.
	if response.StatusCode != http.StatusOK {
		t.Errorf(`want "%d" status code when the email sending operation succeeds, got "%d"`, http.StatusOK, response.StatusCode)
	}

	// Assert that the returned body is the correct one.
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("unable to read the response body after sending an emai: %s", err)
	}

	expectedMessageBody := `{"message":"success"}`
	if string(responseBody) != expectedMessageBody {
		t.Errorf(`unexpected response body. Want "%s", got "%s"`, expectedMessageBody, string(responseBody))
	}

	// Assert that we did not accidentally modify the "from address".
	if overriddenFromAddress != sender.FromAddress {
		t.Errorf(`the "from address" should have been overriden. Want "%s", got "%s"`, overriddenFromAddress, sender.FromAddress)
	}

	// Assert that only one email "was sent".
	if 1 != len(sender.Emails) {
		t.Errorf("want 1 email to be sent, got %d sent", len(sender.Emails))
	}

	// Assert that the sent email contains the expected fields.
	sentEmail := sender.Emails[0]

	// Assert that the subject is correct.
	if email.Subject != sentEmail.Subject {
		t.Errorf(`unexpected email subject sent. Want "%s", got "%s"`, email.Subject, sentEmail.Subject)
	}

	// Assert that the body is correct.
	if email.Body != sentEmail.Body {
		t.Errorf(`unexpected email body sent. Want "%s", got "%s"`, email.Body, sentEmail.Body)
	}

	// Assert that the recipients are correct.
	email.Recipients[1] = "a@mocked.biz"
	if !slicesAreEqual(t, email.Recipients, sentEmail.Recipients) {
		t.Error(`unexpected recipients specified in the sent email`)
	}

	// Assert that the CC recipients are correct.
	email.CcList[1] = "b@mocked.biz"
	if !slicesAreEqual(t, email.CcList, sentEmail.CcList) {
		t.Error(`unexpected CC recipients specified in the sent email`)
	}

	// Assert that the BCC recipients are correct.
	email.BccList[1] = "c@mocked.biz"
	if !slicesAreEqual(t, email.BccList, sentEmail.BccList) {
		t.Error(`unexpected BCC recipients specified in the sent email`)
	}

	// Assert that the specified body type is correct.
	if email.BodyType != sentEmail.BodyType {
		t.Errorf(`unexpected email body type sent. Want "%s", got "%s"`, email.BodyType, sentEmail.BodyType)
	}
}

// TestSendEmailsSkipUsersResolution tests that the incoming emails are correctly parsed and processed, and that the
// output emails to be sent to the mailers contain the incoming email's data. It skips the user resolution and asserts
// that the recipients are left untouched.
func TestSendEmailsSkipUsersResolution(t *testing.T) {
	// Mock the incoming payload.
	recipients := []string{"a@redhat.com", "a"}
	ccList := []string{"copy@redhat.com", "b"}
	bccList := []string{"hiddenCopy@redhat.com", "c"}

	email := &models.Email{
		Subject:    "subject",
		Body:       "body",
		Recipients: recipients,
		CcList:     ccList,
		BccList:    bccList,
		BodyType:   "html",
	}

	emails := &models.Emails{
		Emails:              []models.Email{*email},
		SkipUsersResolution: true,
	}

	// Marshal the mocked body.
	requestBody, err := json.Marshal(emails)
	if err != nil {
		t.Fatalf("unable to marshal the emails model to JSON: %s", err)
	}

	// Mock the request and the response writer.
	request := httptest.NewRequest(http.MethodPost, "/v1/sendEmails", bytes.NewBuffer(requestBody))
	writer := httptest.NewRecorder()

	// Change the mailer module to the "mock" one, and make sure that it gets reverted after the test.
	originalMailerModule := config.Get().MailerModule
	defer func() {
		config.Get().MailerModule = originalMailerModule
	}()
	config.Get().MailerModule = "mock"

	// Use the "mock emailer" module for the test.
	sender := &mailer.MockEmailer{}

	// Change the users' module to "mock", and then rever the value to the original one after the test.
	originalUsersModule := config.Get().UsersModule
	defer func() {
		config.Get().UsersModule = originalUsersModule
	}()
	config.Get().UsersModule = "mock"

	// Call the function under test.
	sendEmails(writer, request, sender)

	// Assert that the operation succeeded.
	response := writer.Result()
	defer response.Body.Close()

	// Assert that we are returning the expected status code.
	if response.StatusCode != http.StatusOK {
		t.Errorf(`want "%d" status code when the email sending operation succeeds, got "%d"`, http.StatusOK, response.StatusCode)
	}

	// Assert that the returned body is the correct one.
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("unable to read the response body after sending an emai: %s", err)
	}

	expectedMessageBody := `{"message":"success"}`
	if string(responseBody) != expectedMessageBody {
		t.Errorf(`unexpected response body. Want "%s", got "%s"`, expectedMessageBody, string(responseBody))
	}

	// Assert that we did not accidentally modify the "from address".
	if config.Get().FromEmail != sender.FromAddress {
		t.Errorf(`the "from address" email should not have been modified by the test. Want "%s", got "%s"`, config.Get().FromEmail, sender.FromAddress)
	}

	// Assert that only one email "was sent".
	if 1 != len(sender.Emails) {
		t.Errorf("want 1 email to be sent, got %d sent", len(sender.Emails))
	}

	// Assert that the sent email contains the expected fields.
	sentEmail := sender.Emails[0]

	// Assert that the subject is correct.
	if email.Subject != sentEmail.Subject {
		t.Errorf(`unexpected email subject sent. Want "%s", got "%s"`, email.Subject, sentEmail.Subject)
	}

	// Assert that the body is correct.
	if email.Body != sentEmail.Body {
		t.Errorf(`unexpected email body sent. Want "%s", got "%s"`, email.Body, sentEmail.Body)
	}

	// Assert that the recipients are correct.
	if !slicesAreEqual(t, email.Recipients, sentEmail.Recipients) {
		t.Error(`unexpected recipients specified in the sent email`)
	}

	// Assert that the CC recipients are correct.
	if !slicesAreEqual(t, email.CcList, sentEmail.CcList) {
		t.Error(`unexpected CC recipients specified in the sent email`)
	}

	// Assert that the BCC recipients are correct.
	if !slicesAreEqual(t, email.BccList, sentEmail.BccList) {
		t.Error(`unexpected BCC recipients specified in the sent email`)
	}

	// Assert that the specified body type is correct.
	if email.BodyType != sentEmail.BodyType {
		t.Errorf(`unexpected email body type sent. Want "%s", got "%s"`, email.BodyType, sentEmail.BodyType)
	}
}

// TestSendEmailsDefaultRecipientFromConfig tests that the incoming emails are correctly parsed and processed, and that
// the  output emails to be sent to the mailers contain the incoming email's data. The default recipient is not
// specified, so it is expected that the default recipient will be grabbed from the configuration.
func TestSendEmailsDefaultRecipientFromConfig(t *testing.T) {
	// Mock the incoming payload.
	email := &models.Email{
		Subject:  "subject",
		Body:     "body",
		BodyType: "html",
	}

	emails := &models.Emails{
		Emails: []models.Email{*email},
	}

	// Marshal the mocked body.
	requestBody, err := json.Marshal(emails)
	if err != nil {
		t.Fatalf("unable to marshal the emails model to JSON: %s", err)
	}

	// Mock the request and the response writer.
	request := httptest.NewRequest(http.MethodPost, "/v1/sendEmails", bytes.NewBuffer(requestBody))
	writer := httptest.NewRecorder()

	// Change the mailer module to the "mock" one, and make sure that it gets reverted after the test.
	originalMailerModule := config.Get().MailerModule
	defer func() {
		config.Get().MailerModule = originalMailerModule
	}()
	config.Get().MailerModule = "mock"

	// Use the "mock emailer" module for the test.
	sender := &mailer.MockEmailer{}

	// Change the users' module to "mock", and then rever the value to the original one after the test.
	originalUsersModule := config.Get().UsersModule
	defer func() {
		config.Get().UsersModule = originalUsersModule
	}()
	config.Get().UsersModule = "mock"

	// Call the function under test.
	sendEmails(writer, request, sender)

	// Assert that the operation succeeded.
	response := writer.Result()
	defer response.Body.Close()

	// Assert that we are returning the expected status code.
	if response.StatusCode != http.StatusOK {
		t.Errorf(`want "%d" status code when the email sending operation succeeds, got "%d"`, http.StatusOK, response.StatusCode)
	}

	// Assert that the returned body is the correct one.
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("unable to read the response body after sending an emai: %s", err)
	}

	expectedMessageBody := `{"message":"success"}`
	if string(responseBody) != expectedMessageBody {
		t.Errorf(`unexpected response body. Want "%s", got "%s"`, expectedMessageBody, string(responseBody))
	}

	// Assert that we did not accidentally modify the "from address".
	if config.Get().FromEmail != sender.FromAddress {
		t.Errorf(`the "from address" email should not have been modified by the test. Want "%s", got "%s"`, config.Get().FromEmail, sender.FromAddress)
	}

	// Assert that only one email "was sent".
	if 1 != len(sender.Emails) {
		t.Errorf("want 1 email to be sent, got %d sent", len(sender.Emails))
	}

	// Assert that the sent email contains the expected fields.
	sentEmail := sender.Emails[0]

	// Assert that the subject is correct.
	if email.Subject != sentEmail.Subject {
		t.Errorf(`unexpected email subject sent. Want "%s", got "%s"`, email.Subject, sentEmail.Subject)
	}

	// Assert that the body is correct.
	if email.Body != sentEmail.Body {
		t.Errorf(`unexpected email body sent. Want "%s", got "%s"`, email.Body, sentEmail.Body)
	}

	// Assert that the recipients are correct.
	if !slicesAreEqual(t, []string{config.Get().ToEmail}, sentEmail.Recipients) {
		t.Error("unexpected default recipient specified. Wanted the default recipient from the configuration.")
	}

	// Assert that the specified body type is correct.
	if email.BodyType != sentEmail.BodyType {
		t.Errorf(`unexpected email body type sent. Want "%s", got "%s"`, email.BodyType, sentEmail.BodyType)
	}
}

// TestSendEmailsDefaultRecipientOverridden tests that the incoming emails are correctly parsed and processed, and that
// the  output emails to be sent to the mailers contain the incoming email's data. The default recipient is specified
// in the incoming payload, so it should be overridden.
func TestSendEmailsDefaultRecipientOverridden(t *testing.T) {
	// Mock the incoming payload.
	email := &models.Email{
		Subject:  "subject",
		Body:     "body",
		BodyType: "html",
	}

	// Override the default recipient.
	defaultRecipient := "default-recipient@redhat.com"
	emails := &models.Emails{
		Emails:           []models.Email{*email},
		DefaultRecipient: defaultRecipient,
	}

	// Marshal the mocked body.
	requestBody, err := json.Marshal(emails)
	if err != nil {
		t.Fatalf("unable to marshal the emails model to JSON: %s", err)
	}

	// Mock the request and the response writer.
	request := httptest.NewRequest(http.MethodPost, "/v1/sendEmails", bytes.NewBuffer(requestBody))
	writer := httptest.NewRecorder()

	// Change the mailer module to the "mock" one, and make sure that it gets reverted after the test.
	originalMailerModule := config.Get().MailerModule
	defer func() {
		config.Get().MailerModule = originalMailerModule
	}()
	config.Get().MailerModule = "mock"

	// Use the "mock emailer" module for the test.
	sender := &mailer.MockEmailer{}

	// Change the users' module to "mock", and then rever the value to the original one after the test.
	originalUsersModule := config.Get().UsersModule
	defer func() {
		config.Get().UsersModule = originalUsersModule
	}()
	config.Get().UsersModule = "mock"

	// Call the function under test.
	sendEmails(writer, request, sender)

	// Assert that the operation succeeded.
	response := writer.Result()
	defer response.Body.Close()

	// Assert that we are returning the expected status code.
	if response.StatusCode != http.StatusOK {
		t.Errorf(`want "%d" status code when the email sending operation succeeds, got "%d"`, http.StatusOK, response.StatusCode)
	}

	// Assert that the returned body is the correct one.
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("unable to read the response body after sending an emai: %s", err)
	}

	expectedMessageBody := `{"message":"success"}`
	if string(responseBody) != expectedMessageBody {
		t.Errorf(`unexpected response body. Want "%s", got "%s"`, expectedMessageBody, string(responseBody))
	}

	// Assert that we did not accidentally modify the "from address".
	if config.Get().FromEmail != sender.FromAddress {
		t.Errorf(`the "from address" email should not have been modified by the test. Want "%s", got "%s"`, config.Get().FromEmail, sender.FromAddress)
	}

	// Assert that only one email "was sent".
	if 1 != len(sender.Emails) {
		t.Errorf("want 1 email to be sent, got %d sent", len(sender.Emails))
	}

	// Assert that the sent email contains the expected fields.
	sentEmail := sender.Emails[0]

	// Assert that the subject is correct.
	if email.Subject != sentEmail.Subject {
		t.Errorf(`unexpected email subject sent. Want "%s", got "%s"`, email.Subject, sentEmail.Subject)
	}

	// Assert that the body is correct.
	if email.Body != sentEmail.Body {
		t.Errorf(`unexpected email body sent. Want "%s", got "%s"`, email.Body, sentEmail.Body)
	}

	// Assert that the recipients are correct.
	if !slicesAreEqual(t, []string{defaultRecipient}, sentEmail.Recipients) {
		t.Error("unexpected default recipient specified. Wanted the overridden default recipient specified in the payload.")
	}

	// Assert that the specified body type is correct.
	if email.BodyType != sentEmail.BodyType {
		t.Errorf(`unexpected email body type sent. Want "%s", got "%s"`, email.BodyType, sentEmail.BodyType)
	}
}
