// Copyright © 2018 Cybozu
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/cybozu-go/neco/generator"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

const defaultIgnoreFile = "artifacts_ignore.yaml"

var rootCmd = &cobra.Command{
	Use:   "generate-artifacts",
	Short: "Generate artifacts.go with the latest release candidates",
	Long: `Generate artifacts.go source code.

This command gathers the latest release candidates of tools used to
build Neco data center, and generates "artifacts.go".

If --release is given, the generated source code will have a build
tag "release".  If not, the generated code will have tag "!release".
`,
	Run: func(cmd *cobra.Command, args []string) {
		data, err := os.ReadFile(*ignoreFile)
		if err != nil && !os.IsNotExist(err) {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(2)
		}

		ignored := &generator.IgnoreConfig{}
		if len(data) > 0 {
			err = yaml.Unmarshal(data, ignored)
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(2)
			}
		}

		cfg := generator.Config{
			Release: *flagRelease,
			Ignored: ignored,
		}
		err = generator.Generate(context.Background(), cfg, os.Stdout)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(2)
		}
	},
}

// Execute executes the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var (
	flagRelease *bool
	ignoreFile  *string
)

func init() {
	flagRelease = rootCmd.Flags().Bool("release", false, "Generate artifacts_prod.go")
	ignoreFile = rootCmd.Flags().String("ignore-file", defaultIgnoreFile, "Filename to ignore artifacts")
}
