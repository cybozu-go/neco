package main

import (
	restrict_pkg "github.com/cybozu-go/neco/pkg/restrictpkg"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(restrict_pkg.RestrictPackageAnalyzer)
}
