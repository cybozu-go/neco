package op

import "github.com/cybozu-go/cke"

// EtcdVolumeName returns etcd volume name
func EtcdVolumeName(e cke.EtcdParams) string {
	if len(e.VolumeName) == 0 {
		return DefaultEtcdVolumeName
	}
	return e.VolumeName
}
