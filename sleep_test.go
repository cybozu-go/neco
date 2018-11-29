package neco

import (
	"context"
	"testing"
	"time"
)

func TestSleepContext(t *testing.T) {
	t.Skip()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	t.Log(time.Now())
	err := SleepContext(ctx, 10*time.Second)
	if err == nil {
		t.Error("err is nil")
	}
	t.Log(time.Now())
}
