//go:build linux

package serf

import (
	"os"
	"testing"
)

const ubuntu = `NAME="Ubuntu"
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

const containerLinux = `NAME="Flatcar Container Linux by Kinvolk"
ID=flatcar
ID_LIKE=coreos
VERSION=2765.2.0
VERSION_ID=2765.2.0
BUILD_ID=2021-03-02-1918
PRETTY_NAME="Flatcar Container Linux by Kinvolk 2765.2.0 (Oklo)"
ANSI_COLOR="38;5;75"
HOME_URL="https://flatcar-linux.org/"
BUG_REPORT_URL="https://issues.flatcar-linux.org"
FLATCAR_BOARD="amd64-usr"
`

func TestGetOSName(t *testing.T) {
	cases := []struct {
		content string
		name    string
	}{
		{content: ubuntu, name: "Ubuntu"},
		{content: containerLinux, name: "Flatcar Container Linux by Kinvolk"},
	}

	for _, c := range cases {
		f, err := os.CreateTemp("", "")
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
		name, err := GetOSName()
		if err != nil {
			t.Fatal(err)
		}
		if name != c.name {
			t.Error("id != c.id:", name)
		}
	}
}

func TestGetOSVersionID(t *testing.T) {
	cases := []struct {
		content string
		id      string
	}{
		{content: ubuntu, id: "18.04"},
		{content: containerLinux, id: "2765.2.0"},
	}

	for _, c := range cases {
		f, err := os.CreateTemp("", "")
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
	f, err := os.CreateTemp("", "")
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
