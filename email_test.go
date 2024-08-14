package main

import "testing"

func TestEmail(t *testing.T) {
	secrets, awsErr := getSecretFromAWS()
	if awsErr != nil {
		t.Errorf("Cannot get secrets from AWS: %s", awsErr.Error())
	}
	body := `Oi mate!
Check out this proper link: https://ppacer.org

Peace out, mate!
	`
	sErr := sendEmail(
		"damians.lbn@gmail.com", "Another test from Go", body, secrets,
	)
	if sErr != nil {
		t.Errorf("Cannot send email: %s", sErr.Error())
	}
}
