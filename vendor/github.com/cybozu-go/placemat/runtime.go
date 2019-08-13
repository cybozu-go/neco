package placemat

import (
	"bufio"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/cybozu-go/log"
)

var vhostNetSupported bool

func init() {
	f, err := os.Open("/proc/modules")
	if err != nil {
		log.Error("failed to open /proc/modules", map[string]interface{}{
			"error": err,
		})
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		if strings.Contains(s.Text(), "vhost_net") {
			vhostNetSupported = true
			return
		}
	}
}

// Runtime contains the runtime information to run Cluster.
type Runtime struct {
	force        bool
	graphic      bool
	enableVirtFS bool
	runDir       string
	ng           nameGenerator
	dataDir      string
	imageCache   *cache
	dataCache    *cache
	sharedDir    string
	tempDir      string
	listenAddr   string
}

// NewRuntime initializes a new Runtime.
func NewRuntime(force, graphic, enableVirtFS bool, runDir, dataDir, cacheDir, sharedDir, listenAddr string) (*Runtime, error) {
	r := &Runtime{
		force:        force,
		graphic:      graphic,
		enableVirtFS: enableVirtFS,
		runDir:       runDir,
		dataDir:      dataDir,
		sharedDir:    sharedDir,
		listenAddr:   listenAddr,
	}

	r.ng.prefix = "pm"

	fi, err := os.Stat(cacheDir)
	switch {
	case err == nil:
		if !fi.IsDir() {
			return nil, errors.New(cacheDir + " is not a directory")
		}
	case os.IsNotExist(err):
		err = os.MkdirAll(cacheDir, 0755)
		if err != nil {
			return nil, err
		}
	default:
		return nil, err
	}

	imageCacheDir := filepath.Join(cacheDir, "image_cache")
	err = os.MkdirAll(imageCacheDir, 0755)
	if err != nil {
		return nil, err
	}

	r.imageCache = &cache{dir: imageCacheDir}

	dataCacheDir := filepath.Join(cacheDir, "data_cache")
	err = os.MkdirAll(dataCacheDir, 0755)
	if err != nil {
		return nil, err
	}

	r.dataCache = &cache{dir: dataCacheDir}

	fi, err = os.Stat(dataDir)
	switch {
	case err == nil:
		if !fi.IsDir() {
			return nil, errors.New(dataDir + " is not a directory")
		}
	case os.IsNotExist(err):
		err = os.MkdirAll(dataDir, 0755)
		if err != nil {
			return nil, err
		}
	default:
		return nil, err
	}

	volumeDir := filepath.Join(dataDir, "volumes")
	err = os.MkdirAll(volumeDir, 0755)
	if err != nil {
		return nil, err
	}

	nvramDir := filepath.Join(dataDir, "nvram")
	err = os.MkdirAll(nvramDir, 0755)
	if err != nil {
		return nil, err
	}

	rktDir := filepath.Join(dataDir, "rkt")
	err = os.MkdirAll(rktDir, 0755)
	if err != nil {
		return nil, err
	}

	tempDir := filepath.Join(dataDir, "temp")
	err = os.MkdirAll(tempDir, 0755)
	if err != nil {
		return nil, err
	}
	myTempDir, err := ioutil.TempDir(tempDir, "")
	if err != nil {
		return nil, err
	}
	r.tempDir = myTempDir

	return r, nil
}

func (r *Runtime) nameGenerator() *nameGenerator {
	return &r.ng
}

func (r *Runtime) socketPath(host string) string {
	return filepath.Join(r.runDir, host+".socket")
}

func (r *Runtime) monitorSocketPath(host string) string {
	return filepath.Join(r.runDir, host+".monitor")
}

func (r *Runtime) guestSocketPath(host string) string {
	return filepath.Join(r.runDir, host+".guest")
}

func (r *Runtime) nvramPath(host string) string {
	return filepath.Join(r.dataDir, "nvram", host+".fd")
}

func (r *Runtime) swtpmSocketDirPath(host string) string {
	return filepath.Join(r.runDir, host)
}

func (r *Runtime) swtpmSocketPath(host string) string {
	return filepath.Join(r.runDir, host, "swtpm.socket")
}
