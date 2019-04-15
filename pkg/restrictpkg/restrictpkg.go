package restrictpkg

import (
	"fmt"
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// RestrictPackageAnalyzer checks if you are using banned package
var RestrictPackageAnalyzer = &analysis.Analyzer{
	Name: "restrictpkg",
	Doc:  doc,
	Run:  run,
	Requires: []*analysis.Analyzer{
		inspect.Analyzer,
	},
	RunDespiteErrors: false,
}

const doc = "restrictpkg checks if you are using banned packages"

var flagPackages string // -name flag

func init() {
	RestrictPackageAnalyzer.Flags.StringVar(&flagPackages, "packages", flagPackages, "list of packages to be restricted")
}

func run(pass *analysis.Pass) (interface{}, error) {
	ins := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	packages := strings.Split(flagPackages, ",")
	nodeFilter := []ast.Node{
		(*ast.ImportSpec)(nil),
	}
	ins.Preorder(nodeFilter, func(n ast.Node) {
		switch n := n.(type) {
		case *ast.ImportSpec:
			for _, p := range packages {
				if n.Path.Value == fmt.Sprintf(`"%s"`, p) {
					pass.Reportf(n.Pos(), fmt.Sprintf("%s package must not be imported", p))
				}
			}
		}
	})

	return nil, nil
}
