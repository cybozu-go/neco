package cke

import (
	"bytes"
	"context"
	"io/ioutil"
	"path/filepath"
	"text/template"

	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/well"
)

// UpdateResources updates user-defined resources
func UpdateResources(ctx context.Context) error {
	templateParams := make(map[string]string)
	for _, img := range neco.CurrentArtifacts.Images {
		templateParams[img.Name] = img.FullName(false)
	}
	images, err := GetCKEImages()
	if err != nil {
		return err
	}
	for _, img := range images {
		templateParams["cke-"+img.Name] = img.FullName(false)
	}

	for _, filename := range neco.CKEUserResourceFiles {
		content, err := ioutil.ReadFile(filename)
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
