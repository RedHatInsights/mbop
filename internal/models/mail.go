package models

import (
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsTypes "github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

type Emails struct {
	Emails              []Email `json:"emails,omitempty"`
	EmailSender         string  `json:"emailSender,omitempty"`
	DefaultRecipient    string  `json:"defaultRecipient,omitempty"`
	SkipUsersResolution bool    `json:"skipUsersResolution,omitempty"`
}

// taken from the BOP openapi spec
type Email struct {
	Subject    string   `json:"subject,omitempty"`
	Body       string   `json:"body,omitempty"`
	Recipients []string `json:"recipients,omitempty"`
	CcList     []string `json:"ccList,omitempty"`
	BccList    []string `json:"bccList,omitempty"`
	BodyType   string   `json:"bodyType,omitempty"`
}

func (e *Email) GetBody() *awsTypes.Body {
	body := &awsTypes.Body{}

	if strings.ToLower(e.BodyType) == "html" {
		body.Html = &awsTypes.Content{Data: aws.String(e.Body)}
	} else {
		body.Text = &awsTypes.Content{Data: aws.String(e.Body)}
	}

	return body
}
