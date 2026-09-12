package main

import (
	"testing"
	"time"

	"github.com/robfig/cron/v3"
)

// CI also runs this test in an empty chroot, where neither the operating
// system nor the Go installation can supply a timezone database.
func TestEmbeddedTimeZones(t *testing.T) {
	for _, tt := range []struct {
		name   string
		month  time.Month
		offset int
	}{
		{name: "Asia/Shanghai", month: time.January, offset: 8 * 60 * 60},
		{name: "America/New_York", month: time.January, offset: -5 * 60 * 60},
		{name: "America/New_York", month: time.July, offset: -4 * 60 * 60},
	} {
		t.Run(tt.name+"/"+tt.month.String(), func(t *testing.T) {
			location, err := time.LoadLocation(tt.name)
			if err != nil {
				t.Fatal(err)
			}
			_, offset := time.Date(2026, tt.month, 15, 12, 0, 0, 0, time.UTC).In(location).Zone()
			if offset != tt.offset {
				t.Fatalf("UTC offset = %d, want %d", offset, tt.offset)
			}
		})
	}

	schedule, err := cron.ParseStandard("CRON_TZ=Asia/Shanghai 0 1 * * *")
	if err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, time.January, 15, 12, 0, 0, 0, time.UTC)
	want := time.Date(2026, time.January, 15, 17, 0, 0, 0, time.UTC)
	if got := schedule.Next(start); !got.Equal(want) {
		t.Fatalf("next cron execution = %s, want %s", got, want)
	}
}
