package main

import (
	"flag"

	"github.com/cybozu-go/well"
)

var (
	flgConfig = flag.String("config", "/etc/neco/neco-worker.yml", "Configuration file path.")
)

func main() {
	flag.Parse()
	well.LogConfig{}.Apply()
}
