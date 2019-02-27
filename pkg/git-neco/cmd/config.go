package cmd

import (
	"bytes"
	"encoding/json"
	"net/url"
	"path/filepath"
	"strings"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	input "github.com/tcnksm/go-input"
)

var config = struct {
	KintoneURL   string `json:"kintone_url"`
	KintoneToken string `json:"kintone_token"`
	GithubToken  string `json:"github_token"`
	GheURL       string `json:"ghe_url"`
	GheToken     string `json:"ghe_token"`
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

	if err := ask(&config.KintoneURL, "kintone app URL", false, func(s string) error {
		_, err := newAppClient(strings.TrimSpace(s), config.KintoneToken)
		return err
	}); err != nil {
		return err
	}
	if err := ask(&config.KintoneToken, "kintone app token", true, nil); err != nil {
		return err
	}
	if err := ask(&config.GithubToken, "github personal token", true, nil); err != nil {
		return err
	}
	if err := ask(&config.GheURL, "github enterprise URL", false, func(s string) error {
		_, err := url.Parse(strings.TrimSpace(s))
		return err
	}); err != nil {
		return err
	}
	if err := ask(&config.GheToken, "github enterprise personal token", true, nil); err != nil {
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
