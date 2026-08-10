package mailer

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	ses "github.com/aws/aws-sdk-go-v2/service/sesv2"
	sesTypes "github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/redhatinsights/mbop/internal/config"
	l "github.com/redhatinsights/mbop/internal/logger"
	"github.com/redhatinsights/mbop/internal/models"
)

var cfg *aws.Config

func InitConfig() error {
	switch config.Get().MailerModule {
	case "aws":
		config, err := awsConfig.LoadDefaultConfig(context.Background(),
			awsConfig.WithRegion(config.Get().SESRegion),
			awsConfig.WithCredentialsProvider(credentials.StaticCredentialsProvider{
				Value: aws.Credentials{
					AccessKeyID:     config.Get().SESAccessKey,
					SecretAccessKey: config.Get().SESSecretKey,
					Source:          "fedrampbop",
				},
			},
			))
		if err != nil {
			return err
		}

		cfg = &config
	case "print":
		l.Log.Info("using printer mailer module")
	default:
		return fmt.Errorf("unsupported mailer module: %v", config.Get().MailerModule)
	}

	return nil
}

var _ = (Emailer)(&awsSESEmailer{})

type awsSESEmailer struct {
	client *ses.Client
}

func (s *awsSESEmailer) SendEmail(ctx context.Context, email *models.Email, fromAddress string) error {
	out, err := s.client.SendEmail(ctx, &ses.SendEmailInput{
		FromEmailAddress: aws.String(fromAddress),
		Destination: &sesTypes.Destination{
			ToAddresses:  email.Recipients,
			CcAddresses:  email.CcList,
			BccAddresses: email.BccList,
		},
		Content: &sesTypes.EmailContent{
			Simple: &sesTypes.Message{
				Subject: &sesTypes.Content{Data: aws.String(email.Subject)},
				Body:    email.GetBody(),
			}},
	})
	if err != nil {
		return err
	}

	l.Log.Info("Sent message successfully, msg id: ", "id", out.MessageId)
	return nil
}
