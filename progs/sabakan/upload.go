package sabakan

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	sabakan "github.com/cybozu-go/sabakan/v2"
	sabac "github.com/cybozu-go/sabakan/v2/client"
	"github.com/cybozu-go/well"
)

const (
	imageOS = "coreos"
)

const retryCount = 5

// UploadContents upload contents to sabakan
func UploadContents(ctx context.Context, sabakanHTTP *http.Client, proxyHTTP *http.Client, version string, auth *DockerAuth, st storage.Storage) error {
	client, err := sabac.NewClient(neco.SabakanLocalEndpoint, sabakanHTTP)
	if err != nil {
		return err
	}

	env := well.NewEnvironment(ctx)
	env.Go(func(ctx context.Context) error {
		return uploadOSImages(ctx, client, proxyHTTP)
	})
	env.Go(func(ctx context.Context) error {
		return uploadAssets(ctx, client, auth)
	})
	env.Stop()
	err = env.Wait()
	if err != nil {
		return err
	}

	// ignitions refers assets, so upload ignitions at the end
	return uploadIgnitions(ctx, client, version, st)
}

// uploadOSImages uploads CoreOS images
func uploadOSImages(ctx context.Context, c *sabac.Client, p *http.Client) error {
	index, err := c.ImagesIndex(ctx, imageOS)
	if err != nil {
		return err
	}

	version := neco.CurrentArtifacts.CoreOS.Version
	if len(index) != 0 && index[len(index)-1].ID == version {
		// already uploaded
		return nil
	}

	kernelURL, initrdURL := neco.CurrentArtifacts.CoreOS.URLs()

	kernelFile, err := ioutil.TempFile("", "kernel")
	if err != nil {
		return err
	}
	defer func() {
		kernelFile.Close()
		os.Remove(kernelFile.Name())
	}()
	initrdFile, err := ioutil.TempFile("", "initrd")
	if err != nil {
		return err
	}
	defer func() {
		initrdFile.Close()
		os.Remove(initrdFile.Name())
	}()

	env := well.NewEnvironment(ctx)

	var kernelSize int64
	env.Go(func(ctx context.Context) error {
		return neco.RetryWithSleep(ctx, retryCount, 10*time.Second,
			func(ctx context.Context) error {
				err := kernelFile.Truncate(0)
				if err != nil {
					return err
				}
				_, err = kernelFile.Seek(0, 0)
				if err != nil {
					return err
				}
				kernelSize, err = downloadFile(ctx, p, kernelURL, kernelFile)
				return err
			},
			func(err error) {
				log.Warn("sabakan: failed to fetch Container Linux kernel", map[string]interface{}{
					log.FnError: err,
					"url":       kernelURL,
				})
			},
		)
	})

	var initrdSize int64
	env.Go(func(ctx context.Context) error {
		return neco.RetryWithSleep(ctx, retryCount, 10*time.Second,
			func(ctx context.Context) error {
				err := initrdFile.Truncate(0)
				if err != nil {
					return err
				}
				_, err = initrdFile.Seek(0, 0)
				if err != nil {
					return err
				}
				initrdSize, err = downloadFile(ctx, p, initrdURL, initrdFile)
				return err
			},
			func(err error) {
				log.Warn("sabakan: failed to fetch Container Linux initrd", map[string]interface{}{
					log.FnError: err,
					"url":       initrdURL,
				})
			},
		)
	})
	env.Stop()
	err = env.Wait()
	if err != nil {
		return err
	}

	_, err = kernelFile.Seek(0, 0)
	if err != nil {
		return err
	}
	_, err = initrdFile.Seek(0, 0)
	if err != nil {
		return err
	}

	return c.ImagesUpload(ctx, imageOS, version, kernelFile, kernelSize, initrdFile, initrdSize)
}

// uploadAssets uploads assets
func uploadAssets(ctx context.Context, c *sabac.Client, auth *DockerAuth) error {
	// Upload bird and chrony with ubuntu-debug
	for _, img := range neco.SystemContainers {
		err := UploadImageAssets(ctx, img, c, auth)
		if err != nil {
			return err
		}
	}

	// Upload other images
	env := well.NewEnvironment(ctx)
	for _, name := range neco.SabakanImages {
		img, err := neco.CurrentArtifacts.FindContainerImage(name)
		if err != nil {
			return err
		}
		env.Go(func(ctx context.Context) error {
			return UploadImageAssets(ctx, img, c, auth)
		})
	}
	env.Stop()
	err := env.Wait()
	if err != nil {
		return err
	}

	// Upload assets for worker node
	files, err := ioutil.ReadDir(neco.WorkerAssetsPath)
	if err != nil {
		return err
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		_, err = c.AssetsUpload(ctx, file.Name(), filepath.Join(neco.WorkerAssetsPath, file.Name()), nil)
		if err != nil {
			return err
		}
	}

	return nil
}

func uploadSystemImageAssets(ctx context.Context, img neco.ContainerImage, c *sabac.Client) error {
	name := neco.ACIAssetName(img)
	need, err := needAssetUpload(ctx, name, c)
	if err != nil {
		return err
	}
	if !need {
		return nil
	}

	_, err = c.AssetsUpload(ctx, name, neco.SystemImagePath(img), nil)
	return err
}

// UploadImageAssets upload docker container image as sabakan assets.
func UploadImageAssets(ctx context.Context, img neco.ContainerImage, c *sabac.Client, auth *DockerAuth) error {
	name := neco.ImageAssetName(img)
	need, err := needAssetUpload(ctx, name, c)
	if err != nil {
		return err
	}
	if !need {
		return nil
	}

	d, err := ioutil.TempDir("", "")
	if err != nil {
		return err
	}
	defer os.RemoveAll(d)

	archive := filepath.Join(d, name)
	err = fetchDockerImageAsArchive(ctx, img, archive, auth)
	if err != nil {
		return err
	}

	_, err = c.AssetsUpload(ctx, name, archive, nil)
	return err
}

func needAssetUpload(ctx context.Context, name string, c *sabac.Client) (bool, error) {
	_, err := c.AssetsInfo(ctx, name)
	if err == nil {
		return false, nil
	}
	if sabac.IsNotFound(err) {
		return true, nil
	}
	return false, err
}

// UploadIgnitions updates ignitions from local file
func UploadIgnitions(ctx context.Context, c *http.Client, id string, st storage.Storage) error {
	client, err := sabac.NewClient(neco.SabakanLocalEndpoint, c)
	if err != nil {
		return err
	}

	return uploadIgnitions(ctx, client, id, st)
}

func uploadIgnitions(ctx context.Context, c *sabac.Client, id string, st storage.Storage) error {
	roles, err := getInstalledRoles()
	if err != nil {
		return err
	}

	_, err = st.GetQuayPassword(ctx)
	if err != nil && err != storage.ErrNotFound {
		return err
	}
	hasSecret := err == nil

	metadata := make(map[string]interface{})
	for _, img := range neco.CurrentArtifacts.Images {
		metadata[img.Name+".img"] = neco.ImageAssetName(img)
		metadata[img.Name+".aci"] = neco.ACIAssetName(img)
		metadata[img.Name+".ref"] = img.FullName(hasSecret)
	}
	for _, img := range neco.SystemContainers {
		metadata[img.Name+".img"] = neco.ImageAssetName(img)
		metadata[img.Name+".aci"] = neco.ACIAssetName(img)
		metadata[img.Name+".ref"] = img.FullName(hasSecret)
	}

	err = setCKEMetadata(metadata)
	if err != nil {
		return err
	}

	pubkey, err := st.GetSSHPubkey(ctx)
	switch err {
	case storage.ErrNotFound:
	case nil:
		metadata["authorized_key"] = pubkey
	default:
		return err
	}

	proxy, err := st.GetProxyConfig(ctx)
	switch err {
	case storage.ErrNotFound:
		metadata["proxy_url"] = ""
	case nil:
		metadata["proxy_url"] = proxy
	default:
		return err
	}

	// set boot server addresses in metadata
	req, err := st.GetRequest(ctx)
	if err != nil {
		return err
	}
	bootServers := make([]string, len(req.Servers))
	for i, lrn := range req.Servers {
		mcs, err := c.MachinesGet(ctx, map[string]string{
			"rack": strconv.Itoa(lrn),
			"role": "boot",
		})
		if err != nil {
			return fmt.Errorf("failed to find boot server in rack %d: %v", lrn, err)
		}
		if len(mcs) != 1 {
			return fmt.Errorf("boot server in rack %d not found", lrn)
		}
		bootServers[i] = mcs[0].Spec.IPv4[0]
	}
	metadata["boot_servers"] = bootServers

	metadata["version"] = req.Version

	for _, role := range roles {
		path := filepath.Join(neco.IgnitionDirectory, "roles", role, "site.yml")

		tmpl, err := sabac.BuildIgnitionTemplate(path, metadata)
		if err != nil {
			return err
		}

		need, err := needIgnitionUpdate(ctx, c, role, id)
		if err != nil {
			return err
		}
		if !need {
			continue
		}
		err = c.IgnitionsSet(ctx, role, id, tmpl)
		if err != nil {
			return err
		}
	}
	return nil
}

func needIgnitionUpdate(ctx context.Context, c *sabac.Client, role, id string) (bool, error) {
	ids, err := c.IgnitionsListIDs(ctx, role)
	if err != nil {
		return false, err
	}

	if len(ids) == 0 {
		return true, nil
	}

	latest := ids[len(ids)-1]
	return latest != id, nil
}

func getInstalledRoles() ([]string, error) {
	paths, err := filepath.Glob(filepath.Join(neco.IgnitionDirectory, "roles", "*", "site.yml"))
	if err != nil {
		return nil, err
	}
	for i, path := range paths {
		paths[i] = filepath.Base(filepath.Dir(path))
	}
	return paths, nil
}

func downloadFile(ctx context.Context, p *http.Client, url string, w io.Writer) (int64, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}
	req = req.WithContext(ctx)
	resp, err := p.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.ContentLength <= 0 {
		return 0, errors.New("unknown content-length")
	}
	return io.Copy(w, resp.Body)
}

func setCKEMetadata(metadata map[string]interface{}) error {
	output, err := exec.Command(neco.CKECLIBin, "images").Output()
	if err != nil {
		return err
	}

	sc := bufio.NewScanner(bytes.NewReader(output))
	for sc.Scan() {
		img, err := neco.ParseContainerImageName(sc.Text())
		if err != nil {
			return err
		}
		metadata["cke:"+img.Name+".img"] = neco.ImageAssetName(img)
		metadata["cke:"+img.Name+".ref"] = img.FullName(false)
	}
	return sc.Err()
}

// UploadDHCPJSON uploads dhcp.json
func UploadDHCPJSON(ctx context.Context, sabakanHTTP *http.Client) error {
	saba, err := sabac.NewClient(neco.SabakanLocalEndpoint, sabakanHTTP)
	if err != nil {
		return err
	}

	f, err := os.Open(neco.SabakanDHCPJSONFile)
	if err != nil {
		return err
	}
	defer f.Close()

	var conf sabakan.DHCPConfig
	err = json.NewDecoder(f).Decode(&conf)
	if err != nil {
		return err
	}

	return saba.DHCPConfigSet(ctx, &conf)
}
