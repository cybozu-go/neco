package teleport

import (
	"io"

	"github.com/cybozu-go/neco"
)

// GenerateConf generates teleport.yaml.template from template.
func GenerateConf(w io.Writer, mylrn int) error {
	return confTmpl.Execute(w, struct {
		AdvertiseIP string
	}{
		AdvertiseIP: neco.BootNode0IP(mylrn).String(),
	})

}
