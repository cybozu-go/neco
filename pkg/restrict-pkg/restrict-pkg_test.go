package restrict_pkg_test

import (
	"testing"

	"github.com/cybozu-go/neco/pkg/restrict-pkg"
	"golang.org/x/tools/go/analysis/analysistest"
)

func Test(t *testing.T) {
	testdata := analysistest.TestData()
	analyzer := restrict_pkg.Analyzer
	err := analyzer.Flags.Set("packages", "html/template")
	if err != nil {
		panic(err)
	}
	analysistest.Run(t, testdata, analyzer, "a", "b")
}
