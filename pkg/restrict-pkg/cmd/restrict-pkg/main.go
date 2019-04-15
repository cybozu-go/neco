package main

import (
	"github.com/cybozu-go/neco/pkg/restrict-pkg"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(restrict_pkg.Analyzer)
}
