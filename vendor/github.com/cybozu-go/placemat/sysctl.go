package placemat

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const sysctlDir = "/proc/sys"

func sysctlPath(name string) string {
	return filepath.Join(sysctlDir, strings.Replace(name, ".", "/", -1))
}

// sysctlGet get the value of the named parameter from "/proc/sys".
//
// If this returns a non-nil error and os.IsNotExist returns true
// for the error, the parameter is not available on this system.
func sysctlGet(name string) (string, error) {
	f, err := os.Open(sysctlPath(name))
	if err != nil {
		return "", err
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// sysctlSet the value of the named parameter.
//
// If this returns a non-nil error and os.IsNotExist returns true
// for the error, the parameter is not available on this system.
func sysctlSet(name, value string) error {
	f, err := os.OpenFile(sysctlPath(name), os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(value)
	return err
}
