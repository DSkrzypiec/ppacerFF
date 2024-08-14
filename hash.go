package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"time"
)

const TimestampFormat = "2006-01-02T15:04:05.999999MST-07:00"

func userHash(email string, now time.Time) string {
	nowStr := now.Format(TimestampFormat)
	var buff bytes.Buffer
	buff.WriteString(email)
	buff.WriteString(nowStr)

	hash := sha256.Sum256(buff.Bytes())
	return fmt.Sprintf("%x", hash)[:24]
}
