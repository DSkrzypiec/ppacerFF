package main

// Copyright 2023 The ppacer Authors.
// Licensed under the Apache License, Version 2.0.
// See LICENSE file in the project root for full license information.

import (
	"fmt"
	"math/rand"
	"time"
)

const (
	// Date format for time.Time serialization and deserialization.
	DateFormat = "2006-01-02"

	// String identifier for local time zone in standard time package.
	LocalTimezoneName = "Local"

	// Time format used on the UI.
	UiTimeFormat = "15:04:05"

	// As UiTimeFormat but including sub-second information.
	UiTimeDetailedFormat = "15:04:05.999999"

	// Date format used on the UI.
	UiDateFormat = "2006-01-02"

	// Timestamp format used on the UI for non-today timestamps.
	UiTimestampFormat = "2006-01-02 15:04:05"
)

var (
	// Timezone used Now and other functions which initialize time.Time.
	ppacerTimezone *time.Location = time.Local

	// In case we would need to mock time.Now() in our test suite.
	timeNow func() time.Time = time.Now
)

// CurrentTz returns currently configured, on package level, timezone.
func CurrentTz() *time.Location {
	return ppacerTimezone
}

// SetTimezone sets package-level timezone. By default it's time.Local. If
// incorrect timezone name is provided, then non-nil error is returned and the
// timezone is not overwritten.
func SetTimezone(timezoneName string) error {
	location, parseErr := time.LoadLocation(timezoneName)
	if parseErr != nil {
		return fmt.Errorf("couldn't parse timezone based on name %s: %w",
			timezoneName, parseErr)
	}
	ppacerTimezone = location
	return nil
}

// Now return the current time in ppacer timezone.
func Now() time.Time {
	return timeNow().In(ppacerTimezone)
}

// ToString serialize given time.Time to string based on TimestampFormat format.
func ToString(t time.Time) string {
	return t.Format(TimestampFormat)
}

// ToStringUI serialize given time.Time to string based on Ui* formats. That
// string is meant to be placed on the UI.
func ToStringUI(t time.Time) string {
	now := time.Now()
	if t.Year() == now.Year() && t.YearDay() == now.YearDay() {
		return t.Format(UiTimeFormat)
	}
	return t.Format(UiTimestampFormat)
}

// ToDateUTCString move given time.Time to UTC location and serialize it to
// date string based on DateFormat format.
func ToDateUTCString(t time.Time) string {
	return t.UTC().Format(DateFormat)
}

// FromString tries to recreate time.Time based on given string value according
// to TimestampFormat format.
func FromString(s string) (time.Time, error) {
	return time.Parse(TimestampFormat, s)
}

// In most cases FromString should be called on strings created by ToString and
// should succeed. In cases when we are pretty sure that FromString will
// succeed, we can use FromStringMust. If FromString would fail for given
// input, error would be logged and time.Time{} would be returned.
func FromStringMust(s string) time.Time {
	t, err := FromString(s)
	if err != nil {
		// TODO(dskrzypiec): should we panic in this case?
		return time.Time{}
	}
	return t
}

func RandomUtcTime(minYear int) time.Time {
	year := rand.Intn(2023-minYear) + minYear
	month := rand.Intn(12) + 1
	day := rand.Intn(28) + 1

	hour := rand.Intn(24)
	minute := rand.Intn(60)
	second := rand.Intn(60)
	ns := rand.Intn(10000000) * 1000

	return time.Date(year, time.Month(month), day, hour, minute, second, ns,
		time.UTC)
}
