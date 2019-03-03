package placemat

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
)

type cache struct {
	dir string
}

func escapeKey(key string) string {
	h := sha256.New()
	h.Write([]byte(key))
	return hex.EncodeToString(h.Sum(nil))
}

func (c *cache) Put(key string, data io.Reader) error {
	ek := escapeKey(key)
	f, err := ioutil.TempFile(c.dir, ".tmp")
	if err != nil {
		return err
	}
	dstName := f.Name()
	defer func() {
		if f != nil {
			f.Close()
		}
		os.Remove(dstName)
	}()

	_, err = io.Copy(f, data)
	if err != nil {
		return err
	}
	err = f.Sync()
	if err != nil {
		return err
	}

	f.Close()
	f = nil

	return os.Rename(dstName, filepath.Join(c.dir, ek))
}

func (c *cache) Get(key string) (io.ReadCloser, error) {
	return os.Open(c.Path(key))
}

func (c *cache) Contains(key string) bool {
	_, err := os.Stat(c.Path(key))
	return !os.IsNotExist(err)
}

func (c *cache) Path(key string) string {
	ek := escapeKey(key)
	return filepath.Join(c.dir, ek)
}

func downloadData(ctx context.Context, u *url.URL, decomp Decompressor, c *cache) error {
	urlString := u.String()

	if c.Contains(urlString) {
		return nil
	}

	req, err := http.NewRequest("GET", urlString, nil)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)

	client := &well.HTTPClient{
		Client:   &http.Client{},
		Severity: log.LvDebug,
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download: %s: %s", res.Status, urlString)
	}

	size, err := strconv.Atoi(res.Header.Get("Content-Length"))
	if err != nil {
		return err
	}

	log.Info("Downloading data...", map[string]interface{}{
		"url":  urlString,
		"size": size,
	})

	var src io.Reader = res.Body
	if decomp != nil {
		newSrc, err := decomp.Decompress(res.Body)
		if err != nil {
			return err
		}
		src = newSrc
	}

	return c.Put(urlString, src)
}

func copyDownloadedData(u *url.URL, dest string, c *cache) error {
	r, err := c.Get(u.String())
	if err != nil {
		return err
	}
	defer r.Close()

	d, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer d.Close()
	_, err = io.Copy(d, r)
	if err != nil {
		return err
	}
	return d.Sync()
}
