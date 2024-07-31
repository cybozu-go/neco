package necorebooter

import (
	"sort"
	"testing"
	"time"

	"github.com/cybozu-go/neco"
	"github.com/robfig/cron/v3"
)

func TestIsWithinSchedule(t *testing.T) {
	cases := []struct {
		name       string
		cronString string
		now        time.Time
		tz         *time.Location
		expected   bool
	}{
		{
			name:       "test1",
			cronString: "* 0 * * *",
			tz:         time.UTC,
			now:        time.Date(2023, 12, 31, 23, 59, 0, 0, time.UTC), // 2023-12-31 23:59:00
			expected:   true,
		},
		{
			name:       "test2",
			cronString: "* 0 * * *",
			tz:         time.UTC,
			now:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), // 2024-01-01 00:00:00
			expected:   true,
		},
		{
			name:       "test3",
			cronString: "* 0 * * *",
			tz:         time.UTC,
			now:        time.Date(2024, 1, 1, 0, 58, 59, 0, time.UTC), // 2024-01-01 00:58:59
			expected:   true,
		},
		{
			name:       "test4",
			cronString: "* 0 * * *",
			tz:         time.UTC,
			now:        time.Date(2024, 1, 1, 0, 59, 0, 0, time.UTC), // 2024-01-01 00:59:00
			expected:   false,
		},
		{
			name:       "test5",
			cronString: "* 0 * * *",
			tz:         time.UTC,
			now:        time.Date(2024, 1, 1, 1, 00, 0, 0, time.UTC), // 2024-01-01 01:00:00
			expected:   false,
		},
		{
			name:       "test6",
			cronString: "0-30 0 * * *",
			tz:         time.UTC,
			now:        time.Date(2023, 12, 31, 23, 59, 0, 0, time.UTC), // 2023-12-31 23:59:00
			expected:   true,
		},
		{
			name:       "test7",
			cronString: "0-30 0 * * *",
			tz:         time.UTC,
			now:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), // 2024-01-01 00:00:00
			expected:   true,
		},
		{
			name:       "test8",
			cronString: "0-30 0 * * *",
			tz:         time.UTC,
			now:        time.Date(2024, 1, 1, 0, 29, 0, 0, time.UTC), // 2024-01-01 00:29:00
			expected:   true,
		},
		{
			name:       "test9",
			cronString: "0-30 0 * * *",
			tz:         time.UTC,
			now:        time.Date(2024, 1, 1, 0, 30, 0, 0, time.UTC), // 2024-01-01 00:30:00
			expected:   false,
		},
		{
			name:       "test10",
			cronString: "* 0 * * 1-5",
			tz:         time.UTC,
			now:        time.Date(2024, 1, 4, 23, 59, 0, 0, time.UTC), // 2024-01-04(Thu) 23:59:00
			expected:   true,
		},
		{
			name:       "test11",
			cronString: "* 0 * * 1-5",
			tz:         time.UTC,
			now:        time.Date(2024, 1, 5, 23, 59, 0, 0, time.UTC), // 2024-01-05(Fri) 23:59:00
			expected:   false,
		},
		{
			name:       "test12",
			cronString: "* 0 * * 1-5",
			tz:         time.UTC,
			now:        time.Date(2024, 1, 6, 0, 0, 0, 0, time.UTC), // 2024-01-06(Sat) 0:00:00
			expected:   false,
		},
		{
			name:       "test13",
			cronString: "* 9 * * 1-5",
			tz:         time.FixedZone("Asia/Tokyo", 9*60*60),
			now:        time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC), // 2024-01-06(Fri) 0:00:00(UTC) = 2024-01-05(Fri) 09:00:00(JST)
			expected:   true,
		},
		{
			name:       "test14",
			cronString: "* 10 * * 1-5",
			tz:         time.FixedZone("Asia/Tokyo", 9*60*60),
			now:        time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC), // 2024-01-06(Fri) 0:00:00(UTC) = 2024-01-05(Fri) 09:00:00(JST)
			expected:   false,
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			schedule, err := cron.ParseStandard(tt.cronString)
			if err != nil {
				t.Fatal(err)
			}
			timeNowFunc = func() time.Time {
				return tt.now
			}
			actual := isWithinSchedule(schedule, tt.tz)
			if actual != tt.expected {
				t.Errorf("unexpected result: want=%v, got=%v", tt.expected, actual)
			}
		})
	}
}

func TestGetAllGroups(t *testing.T) {
	rl := []*neco.RebootListEntry{
		{
			Group: "group1",
		},
		{
			Group: "group2",
		},
		{
			Group: "group2",
		},
		{
			Group: "group3",
		},
	}
	expected := []string{"group1", "group2", "group3"}
	actual := getAllGroups(rl)
	sort.Strings(expected)
	sort.Strings(actual)
	if len(expected) != len(actual) {
		t.Errorf("unexpected result: want=%v, got=%v", expected, actual)
	}
	for i := range expected {
		if expected[i] != actual[i] {
			t.Errorf("unexpected result: want=%v, got=%v", expected, actual)
		}
	}
}

func TestIsEqualContents(t *testing.T) {
	slices1 := []string{"a", "b", "c"}
	slices2 := []string{"b", "a", "c"}
	slices3 := []string{"a", "b", "c", "d"}

	if isEqualContents(slices1, slices2) != true {
		t.Errorf("unexpected result: want=true, got=false")
	}
	if isEqualContents(slices1, slices3) != false {
		t.Errorf("unexpected result: want=false, got=true")
	}
}
