package setup

import (
	"net"
	"testing"
)

func TestBastionIP(t *testing.T) {
	t.Skip()

	ip, err := bastionIP()
	if err != nil {
		t.Fatal(err)
	}

	if !ip.Equal(net.ParseIP("172.16.99.99")) {
		t.Error("bad bastion IP:", ip.String())
	}
}
