package cke

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"text/template"

	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/well"
)

// UpdateResources updates user-defined resources
func UpdateResources(ctx context.Context, st storage.Storage) error {
	templateParams := make(map[string]string)
	lbAddr, err := st.GetLBAddressBlockDefault(ctx)
	templateParams, err = setLBAddress("lbAddressDefault", lbAddr, templateParams, err)
	if err != nil {
		return err
	}
	lbAddr, err = st.GetLBAddressBlockBastion(ctx)
	templateParams, err = setLBAddress("lbAddressBastion", lbAddr, templateParams, err)
	if err != nil {
		return err
	}
	lbAddr, err = st.GetLBAddressBlockInternet(ctx)
	templateParams, err = setLBAddress("lbAddressInternet", lbAddr, templateParams, err)
	if err != nil {
		return err
	}

	env, err := st.GetEnvConfig(ctx)
	if err != nil {
		return err
	}
	var ckeUserResourceFiles []string
	switch env {
	case neco.ProdEnv:
		ckeUserResourceFiles = neco.CKEUserResourceFiles
	case neco.StagingEnv, neco.DevEnv:
		ckeUserResourceFiles = neco.CKEUserResourceFilesPre
	case neco.NoneEnv, neco.TestEnv:
		ckeUserResourceFiles = neco.CKEUserResourceFilesGcp
	}

	for _, filename := range ckeUserResourceFiles {
		content, err := os.ReadFile(filename)
		if err != nil {
			return err
		}

		tmpl, err := template.New(filepath.Base(filename)).Parse(string(content))
		if err != nil {
			return err
		}

		buf := &bytes.Buffer{}
		err = tmpl.Execute(buf, templateParams)
		if err != nil {
			return err
		}

		cmd := well.CommandContext(ctx, neco.CKECLIBin, "resource", "set", "-")
		cmd.Stdin = buf
		err = cmd.Run()
		if err != nil {
			return err
		}
	}
	return nil
}

func setLBAddress(key, lbAddr string, templateParams map[string]string, err error) (map[string]string, error) {
	switch err {
	case storage.ErrNotFound:
		templateParams[key] = ""
	case nil:
		templateParams[key] = lbAddr
	default:
		return nil, err
	}
	return templateParams, nil
}
