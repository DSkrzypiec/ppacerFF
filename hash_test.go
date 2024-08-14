package main

import (
	"testing"
	"time"
)

func TestUserHash(t *testing.T) {
	now := time.Now()
	email1 := "test@gmail.com"

	hash1 := userHash(email1, now)
	hash2 := userHash(email1, now)

	if hash1 != hash2 {
		t.Errorf("Expected the same hash for the same input, got diff: %s vs %s",
			hash1, hash2)
	}

	hash3 := userHash(email1, now.Add(1*time.Millisecond))
	if hash1 == hash3 {
		t.Error("Expected hash3 to be different than hash1, but are the same")
	}
}
