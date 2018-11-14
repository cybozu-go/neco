package worker

import (
	"bytes"
	"io/ioutil"
	"os"
)

func replaceFile(name string, data []byte, mode os.FileMode) (bool, error) {
	current, err := ioutil.ReadFile(name)
	switch {
	case os.IsNotExist(err):
	case err == nil && !bytes.Equal(current, data):
	default:
		return false, err
	}

	f, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Write(data)
	if err != nil {
		return false, err
	}

	err = f.Sync()
	if err != nil {
		return false, err
	}

	return true, nil
}
