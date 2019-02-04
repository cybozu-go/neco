package cmd

import (
	"fmt"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/spf13/cobra"
)

// imageCmd represents the image command
var imageCmd = &cobra.Command{
	Use:   "image NAME",
	Short: "show docker image URL for NAME",
	Long:  `Show docker image URL for NAME.`,

	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		img, err := neco.CurrentArtifacts.FindContainerImage(args[0])
		if err != nil {
			log.ErrorExit(err)
		}

		fmt.Println(img.FullName(false))
	},
}

func init() {
	rootCmd.AddCommand(imageCmd)
}
