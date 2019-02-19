//go:generate statik -f -src=./../../gcp/public -dest=./../../gcp

package main

import (
	_ "github.com/cybozu-go/neco/gcp/statik"
	"github.com/cybozu-go/neco/pkg/necogcp/cmd"
)

func main() {
	cmd.Execute()
}
