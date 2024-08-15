package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	telegramSecretName = "telegram/homeAppDev"
)

type telegramSecrets struct {
	BotToken  string `json:"botToken"`
	ChannelId string `json:"channelId"`
}

type Telegram struct {
	botToken   string
	channelId  int64
	httpClient *http.Client
}

func NewTelegram() *Telegram {
	secret, sErr := getTelegramSecret()
	if sErr != nil {
		log.Panicf("Cannot get Telegram secrets from AWS: %s", sErr.Error())
	}
	channelId, castErr := strconv.ParseInt(secret.ChannelId, 10, 64)
	if castErr != nil {
		log.Panicf("Cannot cast channelId (%s) to int64: %s",
			secret.ChannelId, castErr.Error())
	}
	return &Telegram{
		botToken:   secret.BotToken,
		channelId:  channelId,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (t *Telegram) Send(msg string) error {
	url := t.sendMessageUrl(msg)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, rErr := http.NewRequestWithContext(ctx, "GET", url, nil)
	if rErr != nil {
		return rErr
	}
	resp, err := t.httpClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("expected status %d, got: %d", http.StatusOK,
			resp.StatusCode)
	}
	return nil
}

func getTelegramSecret() (telegramSecrets, error) {
	return getSecretFromAWS[telegramSecrets](telegramSecretName)
}

func (t *Telegram) sendMessageUrl(text string) string {
	const urlTmpl = "https://api.telegram.org/bot%s/sendMessage?chat_id=%d&text=%s"
	encodedText := url.QueryEscape(text)
	return fmt.Sprintf(urlTmpl, t.botToken, t.channelId, encodedText)
}
