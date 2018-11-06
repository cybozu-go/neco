package worker

import (
	"bufio"
	"bytes"
	"context"
	"net/textproto"

	"github.com/cybozu-go/well"
)

// GetDebianVersion returns debian package version.
func GetDebianVersion(pkg string) (string, error) {
	data, err := well.CommandContext(context.Background(), "dpkg", "-s", pkg).Output()
	if err != nil {
		return "", err
	}

	data = append(data, '\n')
	r := textproto.NewReader(bufio.NewReader(bytes.NewReader(data)))
	hdrs, err := r.ReadMIMEHeader()
	if err != nil {
		return "", err
	}
	return hdrs.Get("Version"), nil
}
