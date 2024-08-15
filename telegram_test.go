package main

import "testing"

func TestTelegram(t *testing.T) {
	telegram := NewTelegram()
	err := telegram.Send("test from ppacerFF")
	if err != nil {
		t.Errorf("Error while sending Telegram message: %s", err.Error())
	}
}
