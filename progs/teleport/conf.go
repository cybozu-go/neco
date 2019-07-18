package teleport

import (
	"io"

	"github.com/cybozu-go/neco"
)

// GenerateConfBase generates teleport.yaml.base from template.
func GenerateConfBase(w io.Writer, mylrn int) error {
	return confTmpl.Execute(w, struct {
		AdvertiseIP string
	}{
		AdvertiseIP: neco.BootNode0IP(mylrn).String(),
	})

}
