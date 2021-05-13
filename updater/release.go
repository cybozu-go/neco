package updater

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type necoRelease struct {
	date    time.Time
	version int
}

func newNecoRelease(tag string) (*necoRelease, error) {
	tags := strings.Split(tag, "-")
	if len(tags) != 2 {
		return nil, fmt.Errorf(`tag should have "YYYY.MM.DD-UNIQUE_ID", but got %s`, tag)
	}
	d, err := time.Parse("2006.01.02", tags[0])
	if err != nil {
		return nil, err
	}
	v, err := strconv.Atoi(tags[1])
	if err != nil {
		return nil, err
	}
	return &necoRelease{d, v}, nil
}

func (r necoRelease) isNewerThan(target *necoRelease) bool {
	return !r.date.Before(target.date) && r.version > target.version
}
