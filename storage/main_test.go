package storage

import (
	"os"
	"testing"

	"github.com/cybozu-go/neco/storage/test"
)

func TestMain(m *testing.M) {
	os.Exit(test.RunTestMain(m))
}
