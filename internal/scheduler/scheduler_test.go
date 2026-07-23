package scheduler

import (
	"testing"
	"time"

	"github.com/robfig/cron/v3"
)

func TestCollectUpcomingRunsEveryFifteenMinutes(t *testing.T) {
	location := time.FixedZone("CST", 8*60*60)
	now := time.Date(2026, time.July, 23, 13, 12, 0, 0, location)
	schedule, err := cron.ParseStandard("*/15 13-15 * * 1-5")
	if err != nil {
		t.Fatal(err)
	}

	runs := collectUpcomingRuns(schedule, schedule.Next(now), now.Add(time.Hour), 50)
	want := []time.Time{
		time.Date(2026, time.July, 23, 13, 15, 0, 0, location),
		time.Date(2026, time.July, 23, 13, 30, 0, 0, location),
		time.Date(2026, time.July, 23, 13, 45, 0, 0, location),
		time.Date(2026, time.July, 23, 14, 0, 0, 0, location),
	}
	if len(runs) != len(want) {
		t.Fatalf("got %d runs, want %d: %v", len(runs), len(want), runs)
	}
	for i := range want {
		if !runs[i].Equal(want[i]) {
			t.Errorf("run %d = %v, want %v", i, runs[i], want[i])
		}
	}
}

func TestCollectUpcomingRunsHonorsLimit(t *testing.T) {
	now := time.Date(2026, time.July, 23, 13, 12, 0, 0, time.UTC)
	schedule, err := cron.ParseStandard("* * * * *")
	if err != nil {
		t.Fatal(err)
	}

	runs := collectUpcomingRuns(schedule, schedule.Next(now), now.Add(24*time.Hour), 3)
	if len(runs) != 3 {
		t.Fatalf("got %d runs, want 3", len(runs))
	}
}
