package teleport

import (
	"io"

	"github.com/cybozu-go/neco"
)

// GenerateConf generates teleport.yaml from template.
func GenerateConf(w io.Writer, mylrn int, authToken string) error {
	return confTmpl.Execute(w, struct {
		AdvertiseIP string
		AuthToken   string
	}{
		AdvertiseIP: neco.BootNode0IP(mylrn).String(),
		AuthToken:   authToken,
	})
}
