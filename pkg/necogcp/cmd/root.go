package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco/gcp"
	"github.com/cybozu-go/well"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	cfg     *gcp.Config
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "necogcp",
	Short: "necogcp is GCP management tool for Neco project",
	Long:  `necogcp is GCP management tool for Neco project.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// without this, each subcommand's RunE would display usage text.
		cmd.SilenceUsage = true

		err := well.LogConfig{}.Apply()
		if err != nil {
			return err
		}

		cfg, err = gcp.NewConfig()
		if err != nil {
			return err
		}

		yamlTagOption := func(c *mapstructure.DecoderConfig) {
			c.TagName = "yaml"
		}
		viper.Unmarshal(cfg, yamlTagOption)

		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", filepath.Join(os.Getenv("HOME"), ".necogcp.yml"), "config file")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := homedir.Dir()
		if err != nil {
			log.ErrorExit(err)
		}

		viper.AddConfigPath(home)
		viper.SetConfigName(".necogcp")
		viper.SetConfigType("yml")
	}

	viper.ReadInConfig()
}
