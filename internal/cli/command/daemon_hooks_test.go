package command

import (
	"testing"
	"time"
)

func TestParseDaemonIntervalPattern(t *testing.T) {
	t.Run("duration", func(t *testing.T) {
		got, ok := parseDaemonIntervalPattern("5m")
		if !ok || got != 5*time.Minute {
			t.Fatalf("parse duration = (%v, %v), want (5m, true)", got, ok)
		}
	})

	t.Run("every syntax", func(t *testing.T) {
		got, ok := parseDaemonIntervalPattern("@every 30s")
		if !ok || got != 30*time.Second {
			t.Fatalf("parse @every = (%v, %v), want (30s, true)", got, ok)
		}
	})
}

func TestParseDaemonCronPattern(t *testing.T) {
	t.Run("standard cron", func(t *testing.T) {
		got, ok := parseDaemonCronPattern("0 * * * *")
		if !ok || got != "0 * * * *" {
			t.Fatalf("parse cron = (%q, %v), want (%q, true)", got, ok, "0 * * * *")
		}
	})

	t.Run("descriptor", func(t *testing.T) {
		got, ok := parseDaemonCronPattern("@daily")
		if !ok || got != "@daily" {
			t.Fatalf("parse descriptor = (%q, %v), want (%q, true)", got, ok, "@daily")
		}
	})

	t.Run("duration is not cron", func(t *testing.T) {
		if _, ok := parseDaemonCronPattern("5m"); ok {
			t.Fatal("duration should not be treated as cron")
		}
	})
}
