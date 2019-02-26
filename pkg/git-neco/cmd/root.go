package cmd

import (
	"fmt"
	"os"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "git-neco",
	Short: "git extension for Neco project",
	Long: `"git neco" subcommand for Neco project.

This command read configurations to access GitHub and kintone from
YAML file at $HOME/.git-neco.yml or environment variables using viper.

First time user should run "git neco config" to configure git-neco.`,

	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		err := well.LogConfig{}.Apply()
		if err != nil {
			log.ErrorExit(err)
		}
		cmd.SilenceUsage = true
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

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.git-neco.yaml)")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".git-neco" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".git-neco")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	viper.ReadInConfig()

	decodeToJSON := func(c *mapstructure.DecoderConfig) {
		c.TagName = "json"
	}
	if err := viper.Unmarshal(&config, decodeToJSON); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
