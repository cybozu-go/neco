package cmd

import (
	"errors"
	"net/url"
	"strconv"
	"strings"

	kintone "github.com/kintone/go-kintone"
)

func newAppClient(appURL, token string) (*kintone.App, error) {
	u, err := url.Parse(appURL)
	if err != nil {
		return nil, err
	}

	if !strings.HasPrefix(u.Path, "/k/") {
		return nil, errors.New("invalid kintone app URL: " + appURL)
	}
	parts := strings.Split(u.Path[3:], "/")
	appID, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return nil, errors.New("invalid kintone app ID: " + parts[0])
	}

	return &kintone.App{
		Domain:   u.Host,
		AppId:    appID,
		ApiToken: token,
	}, nil
}
