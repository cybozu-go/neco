package restrict_pkg

import (
	"fmt"
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var Analyzer = &analysis.Analyzer{
	Name: "restrict_pkg",
	Doc:  Doc,
	Run:  run,
	Requires: []*analysis.Analyzer{
		inspect.Analyzer,
	},
	RunDespiteErrors: false,
}

const Doc = "restrict-pkg checks if you are using a banned packages"

var flagPackages string // -name flag

func init() {
	Analyzer.Flags.StringVar(&flagPackages, "packages", flagPackages, "list of packages to be restricted")
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
