package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/smtp"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

const (
	awsRegion       = "eu-central-1"
	from            = "info@dskrzypiec.dev"
	emailSecretName = "email/dskrzypiec/info"
)

type emailSecret struct {
	Host     string `json:"smtpHost"`
	Port     string `json:"smtpPort"`
	Address  string `json:"address"`
	Password string `json:"password"`
}

func sendEmail(to, subject, body string, secrets emailSecret) error {
	message := fmt.Sprintf(`From: %s
To: %s
Subject: %s
MIME-Version: 1.0
Content-Type: text/plain; charset="UTF-8"

%s
	`, from, to, subject, body)

	auth := smtp.PlainAuth("", from, secrets.Password, secrets.Host)
	tlsconfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         secrets.Host,
	}

	// Connect to the SMTP server
	conn, err := tls.Dial("tcp", secrets.Host+":"+secrets.Port, tlsconfig)
	if err != nil {
		return err
	}

	// Create a new SMTP client from the connection
	client, err := smtp.NewClient(conn, secrets.Host)
	if err != nil {
		return err
	}

	// Authenticate
	if err = client.Auth(auth); err != nil {
		return err
	}

	// Set the sender and recipient
	if err = client.Mail(from); err != nil {
		return err
	}
	if err = client.Rcpt(to); err != nil {
		return err
	}

	// Get the data writer and send the email
	writer, err := client.Data()
	if err != nil {
		return err
	}

	_, err = writer.Write([]byte(message))
	if err != nil {
		return err
	}

	err = writer.Close()
	if err != nil {
		return err
	}

	err = client.Quit()
	if err != nil {
		return err
	}
	return nil
}

func getEmailSecrets() (emailSecret, error) {
	return getSecretFromAWS[emailSecret](emailSecretName)
}

func getSecretFromAWS[T any](secretName string) (T, error) {
	var secret T
	cfg, err := config.LoadDefaultConfig(
		context.TODO(), config.WithRegion(awsRegion),
	)
	if err != nil {
		return secret, err
	}

	svc := secretsmanager.NewFromConfig(cfg)
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	}

	result, err := svc.GetSecretValue(context.TODO(), input)
	if err != nil {
		return secret, fmt.Errorf("failed to retrieve secret: %w", err)
	}

	if result.SecretString != nil {
		err = json.Unmarshal([]byte(*result.SecretString), &secret)
		if err != nil {
			return secret,
				fmt.Errorf("failed to unmarshal secret JSON: %v", err)
		}
		return secret, nil
	}
	return secret, fmt.Errorf("secret string is nil")
}
