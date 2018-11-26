// +build linux

package serf

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestGetOSVersionID(t *testing.T) {
	ubuntu := `NAME="Ubuntu"
VERSION="18.04.1 LTS (Bionic Beaver)"
ID=ubuntu
ID_LIKE=debian
PRETTY_NAME="Ubuntu 18.04.1 LTS"
VERSION_ID="18.04"
HOME_URL="https://www.ubuntu.com/"
SUPPORT_URL="https://help.ubuntu.com/"
BUG_REPORT_URL="https://bugs.launchpad.net/ubuntu/"
PRIVACY_POLICY_URL="https://www.ubuntu.com/legal/terms-and-policies/privacy-policy"
VERSION_CODENAME=bionic
UBUNTU_CODENAME=bionic
`
	containerLinux := `NAME="Container Linux by CoreOS"
ID=coreos
VERSION=1911.3.0
VERSION_ID=1911.3.0
BUILD_ID=2018-11-05-1815
PRETTY_NAME="Container Linux by CoreOS 1911.3.0 (Rhyolite)"
ANSI_COLOR="38;5;75"
HOME_URL="https://coreos.com/"
BUG_REPORT_URL="https://issues.coreos.com"
COREOS_BOARD="amd64-usr"
`

	cases := []struct {
		content string
		id      string
	}{
		{content: ubuntu, id: "18.04"},
		{content: containerLinux, id: "1911.3.0"},
	}

	for _, c := range cases {
		f, err := ioutil.TempFile("", "")
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			f.Close()
			os.Remove(f.Name())
		}()

		_, err = f.WriteString(c.content)
		if err != nil {
			t.Fatal(err)
		}
		f.Sync()

		osReleasePath = f.Name()
		id, err := GetOSVersionID()
		if err != nil {
			t.Fatal(err)
		}
		if id != c.id {
			t.Error("id != c.id:", id)
		}
	}
}

func TestGetSerial(t *testing.T) {
	f, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		f.Close()
		os.Remove(f.Name())
	}()

	_, err = f.WriteString("machine-1234abcd\n")
	if err != nil {
		t.Fatal(err)
	}
	f.Sync()

	serialPath = f.Name()
	serial, err := GetSerial()
	if err != nil {
		t.Fatal(err)
	}
	if serial != "machine-1234abcd" {
		t.Error(`serial != "machine-1234abcd"`, serial)
	}
}
