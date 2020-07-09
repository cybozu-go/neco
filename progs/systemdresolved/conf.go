package systemdresolved

import (
	"io"
)

// GenerateConfBase generates neco.conf.base from template.
func GenerateConfBase(w io.Writer, dnsAddress string) error {
	return confTmpl.Execute(w, struct {
		DNSAddress string
	}{
		DNSAddress: dnsAddress,
	})
}
