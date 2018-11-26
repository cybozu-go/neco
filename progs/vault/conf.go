package vault

import (
	"fmt"
	"io"
	"strings"

	"github.com/cybozu-go/neco"
)

// GenerateConf generates vault.hcl from template.
func GenerateConf(w io.Writer, mylrn int, lrns []int) error {
	myip := neco.BootNode0IP(mylrn).String()
	return confTmpl.Execute(w, struct {
		ServerCertFile string
		ServerKeyFile  string
		APIAddr        string
		ClusterAddr    string
		EtcdEndpoints  string
		EtcdCertFile   string
		EtcdKeyFile    string
	}{
		ServerCertFile: neco.ServerCertFile,
		ServerKeyFile:  neco.ServerKeyFile,
		APIAddr:        fmt.Sprintf("https://%s:8200", myip),
		ClusterAddr:    fmt.Sprintf("https://%s:8201", myip),
		EtcdEndpoints:  strings.Join(neco.EtcdEndpoints(lrns), ","),
		EtcdCertFile:   neco.VaultCertFile,
		EtcdKeyFile:    neco.VaultKeyFile,
	})
}
