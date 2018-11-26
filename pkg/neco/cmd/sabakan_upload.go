package cmd

import (
	"context"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco/progs/sabakan"
	"github.com/spf13/cobra"
)

// sabakanUploadCmd implements "sabakan-upload"
var sabakanUploadCmd = &cobra.Command{
	Use:   "sabakan-upload",
	Short: "Upload sabakan contents using artifacts.go",
	Long: `Upload sabakan contents using artifacts.go
If uploaded versions are up to date, do nothing.
`,
	Run: func(cmd *cobra.Command, args []string) {
		err := sabakan.UploadContents(context.Background())
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(sabakanUploadCmd)
}
