package necogcpfunctions

import "time"

var jpHolidays = []string{
	"20210101",
}

func getDateStrInJST() (string, error) {
	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		return "", err
	}
	return time.Now().In(loc).Format("20060102"), nil
}

func isHoliday(target string, holidays []string) bool {
	for _, h := range holidays {
		if target == h {
			return true
		}
	}
	return false
}
