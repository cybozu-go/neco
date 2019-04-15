package restrictpkg_test

import (
	"testing"

	restrict_pkg "github.com/cybozu-go/neco/pkg/restrictpkg"
	"golang.org/x/tools/go/analysis/analysistest"
)

func Test(t *testing.T) {
	testdata := analysistest.TestData()
	analyzer := restrict_pkg.RestrictPackageAnalyzer
	err := analyzer.Flags.Set("packages", "html/template")
	if err != nil {
		panic(err)
	}
	analysistest.Run(t, testdata, analyzer, "a", "b")
}
