package necogcpfunctions

import "testing"

func TestIsHoliday(t *testing.T) {
	testCases := []struct {
		target   string
		holidays []string
		result   bool
	}{
		{"20200101", []string{"20200101", "20200102"}, true},
		{"20200201", []string{"20200101", "20200102"}, false},
	}

	for _, tt := range testCases {
		if isHoliday(tt.target, tt.holidays) != tt.result {
			t.Errorf("%s is holiday", tt.target)
		}
	}
}
