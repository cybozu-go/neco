package main

import (
	"flag"

	"github.com/cybozu-go/well"
)

func main() {
	flag.Parse()
	well.LogConfig{}.Apply()
}
