package neco

import (
	"os"
	"path/filepath"
)

// WriteFile write data to a file.
func WriteFile(filename string, data string) error {
	err := os.MkdirAll(filepath.Dir(filename), 0755)
	if err != nil {
		return err
	}
	return os.WriteFile(filename, []byte(data), 0644)
}
