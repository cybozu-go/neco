package necorebooter

import (
	"slices"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/neco"
	"github.com/robfig/cron/v3"
)

var timeNowFunc = time.Now

func isWithinSchedule(schedule cron.Schedule, timeZone *time.Location) bool {
	now := timeNowFunc().In(timeZone)
	next := schedule.Next(now)
	// If the next scheduled time is within 1 minutesfrom now, we consider that we are within the schedule.
	if next.Sub(now) <= time.Second*60 {
		return true
	} else {
		return false
	}
}

func findRebootQueueEntryFromRebootListEntry(rebootQueueEntries []*cke.RebootQueueEntry, rebootListEntry neco.RebootListEntry) *cke.RebootQueueEntry {
	for _, entry := range rebootQueueEntries {
		if entry.Node == rebootListEntry.Node {
			return entry
		}
	}
	return nil
}

func getAllGroups(rebootListEntries []*neco.RebootListEntry) []string {
	groups := make([]string, 0)
	for _, entry := range rebootListEntries {
		groups = append(groups, entry.Group)
	}
	slices.Sort(groups)
	return slices.Compact(groups)
}
