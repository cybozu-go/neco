package placemat

import (
	"compress/bzip2"
	"compress/gzip"
	"errors"
	"io"
	"os"
)

// Decompressor defines an interface to decompress data from io.Reader.
type Decompressor interface {
	Decompress(io.Reader) (io.Reader, error)
}

type bzip2Decompressor struct{}

func (d bzip2Decompressor) Decompress(r io.Reader) (io.Reader, error) {
	return bzip2.NewReader(r), nil
}

type gzipDecompressor struct{}

func (d gzipDecompressor) Decompress(r io.Reader) (io.Reader, error) {
	return gzip.NewReader(r)
}

// NewDecompressor returns a Decompressor for "format".
// If format is not supported, this returns a non-nil error.
func NewDecompressor(format string) (Decompressor, error) {
	switch format {
	case "bzip2":
		return bzip2Decompressor{}, nil
	case "gzip":
		return gzipDecompressor{}, nil
	case "":
		return nil, nil
	}

	return nil, errors.New("unsupported compression format: " + format)
}

// writeToFile copies contents of file at srcPath to destPath,
// optionally decompressing the source contents if decomp is not nil.
func writeToFile(srcPath, destPath string, decomp Decompressor) error {
	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer f.Close()

	destFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer destFile.Close()

	var src io.Reader = f
	if decomp != nil {
		newSrc, err := decomp.Decompress(src)
		if err != nil {
			return err
		}
		src = newSrc
	}

	_, err = io.Copy(destFile, src)
	return err
}
