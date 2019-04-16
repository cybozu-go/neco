package cmd

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tcnksm/go-input"
)

var config = struct {
	GithubToken string `json:"github_token"`
	ZenhubToken string `json:"zenhub_extension_token"`
}{}

func configInput() error {
	ui := input.DefaultUI()
	ask := func(p *string, query string, mask bool, validate func(s string) error) error {
		ans, err := ui.Ask(query, &input.Options{
			Default:      *p,
			Required:     true,
			Loop:         true,
			MaskDefault:  mask,
			ValidateFunc: validate,
		})
		if err != nil {
			return err
		}
		*p = strings.TrimSpace(ans)
		return nil
	}

	if err := ask(&config.GithubToken, "github personal token", true, nil); err != nil {
		return err
	}

	if err := ask(&config.ZenhubToken, "zenhub extension token", true, nil); err != nil {
		return err
	}

	return nil
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "configure git-neco",
	Long: `Configure git-neco interactively.

The command will save the inputted information in the
configuration file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := configInput(); err != nil {
			return err
		}
		viper.SetConfigType("json")
		data, err := json.Marshal(config)
		if err != nil {
			return err
		}
		if err := viper.ReadConfig(bytes.NewReader(data)); err != nil {
			return err
		}
		filename := viper.ConfigFileUsed()
		if filename == "" {
			home, err := homedir.Dir()
			if err != nil {
				return err
			}
			filename = filepath.Join(home, ".git-neco.yml")
		}
		return viper.WriteConfigAs(filename)
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}
