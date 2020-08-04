package autodctest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"log"

	"cloud.google.com/go/pubsub"
	"github.com/cybozu-go/well"
	"github.com/kelseyhightower/envconfig"
)

const (
	deleteInstancesMode = "delete"
	createInstancesMode = "create"

	machineType = "n1-standard-32"

	skipAutoDeleteLabelKey      = "skip-auto-delete"
	excludeSkipAutoDeleteFilter = "labels." + skipAutoDeleteLabelKey + "!=true"

	startupScriptDir  = "/tmp"
	startupScriptName = "startup.sh"
)

// Body is body of Pub/Sub message.
type Body struct {
	Mode               string `json:"mode"`
	InstanceNamePrefix string `json:"namePrefix"`
	InstancesNum       int    `json:"num"`
	DoForceDelete      bool   `json:"doForce"`
}

// Env is cloud function environment variables
type Env struct {
	ProjectID          string `envconfig:"GCP_PROJECT" required:"true"`
	Zone               string `envconfig:"ZONE" required:"true"`
	ServiceAccountName string `envconfig:"SERVICE_ACCOUNT_NAME" required:"true"`
	AccountJSONPath    string `envconfig:"ACCOUNT_JSON_PATH" required:"true"`
}

// PubSubEntryPoint consumes a Pub/Sub message
func PubSubEntryPoint(ctx context.Context, m *pubsub.Message) error {
	log.Printf("debug: %s", string(m.Data))

	var b Body
	err := json.Unmarshal(m.Data, &b)
	if err != nil {
		log.Printf("error: %v", err)
		return err
	}
	log.Printf("debug: %#v", b)

	var e Env
	err = envconfig.Process("", &e)
	if err != nil {
		log.Printf("error: %v", err)
		return err
	}
	log.Printf("debug: %#v", e)

	switch b.Mode {
	case createInstancesMode:
		log.Printf("create %s %d", b.InstanceNamePrefix, b.InstancesNum)
		today, err := getDateStrInJST()
		if err != nil {
			log.Printf("error: %v", err)
			return nil
		}
		if isHoliday(today, jpHolidays) {
			log.Println("info: today is holiday! skip creating dctest")
			return nil
		}
		return createInstances(
			ctx,
			b.InstancesNum,
			e.ProjectID,
			e.Zone,
			b.InstanceNamePrefix,
			e.ServiceAccountName,
			e.AccountJSONPath,
		)
	case deleteInstancesMode:
		log.Printf("delete %v", b.DoForceDelete)
		return deleteInstances(
			ctx,
			e.ProjectID,
			e.Zone,
			b.DoForceDelete,
		)
	default:
		err := fmt.Errorf("invalid mode was given: %s", b.Mode)
		log.Fatalf("error: %v", err)
		return err
	}
}

func makeInstanceName(prefix string, index int) string {
	return fmt.Sprintf("%s-%d", prefix, index)
}

func fetchInstanceNamesWithFilter(
	ctx context.Context,
	projectID string,
	zone string,
	filter string,
) (map[string]struct{}, error) {
	cmd := []string{
		"gcloud", "compute", "instances", "list",
		"--project", projectID,
		"--zone", zone,
		"--format", "json(name)",
		"--filter", filter,
	}

	buf := new(bytes.Buffer)
	c := well.CommandContext(ctx, cmd[0], cmd[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = buf
	c.Stderr = os.Stderr
	err := c.Run()
	if err != nil {
		return nil, err
	}

	names := []struct {
		Name string `json:"name"`
	}{}
	err = json.Unmarshal(buf.Bytes(), &names)
	if err != nil {
		return nil, err
	}

	set := make(map[string]struct{})
	for _, n := range names {
		set[n.Name] = struct{}{}
	}
	return set, err
}

func createInstances(
	ctx context.Context,
	instancesNum int,
	projectID string,
	zone string,
	instanceNamePrefix string,
	serviceAccountName string,
	accountJSONPath string,
) error {
	instances, err := fetchInstanceNamesWithFilter(
		ctx,
		projectID,
		zone,
		"",
	)
	if err != nil {
		return err
	}

	tmpfilePath, err := createStartupScript(accountJSONPath)
	if err != nil {
		return err
	}
	defer os.Remove(tmpfilePath)

	e := well.NewEnvironment(ctx)
	for i := 0; i < instancesNum; i++ {
		name := makeInstanceName(instanceNamePrefix, i)
		if _, ok := instances[name]; ok {
			log.Printf("info: skip creating %s because it already exists", name)
			continue
		}

		e.Go(func(ctx context.Context) error {
			cmd := []string{
				"gcloud", "compute", "instances", "create", name,
				"--project", projectID,
				"--account", serviceAccountName,
				"--zone", zone,
				"--image", "vmx-enabled",
				"--machine-type", machineType,
				"--local-ssd", "interface=scsi",
				"--metadata-from-file", "startup-script=" + tmpfilePath,
			}
			c := well.CommandContext(ctx, cmd[0], cmd[1:]...)
			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Run()
		})
	}
	well.Stop()
	return well.Wait()
}

func createStartupScript(accountJSONPath string) (string, error) {
	tmp, err := ioutil.TempFile(startupScriptDir, startupScriptName)
	if err != nil {
		return "", err
	}
	defer func() {
		tmp.Close()
	}()

	script := `#! /bin/sh

# Run fluentd to export syslog to Cloud Logging
curl -sSO https://dl.google.com/cloudagents/add-logging-agent-repo.sh
bash add-logging-agent-repo.sh
apt-get update
apt-cache madison google-fluentd
apt-get install -y google-fluentd
apt-get install -y google-fluentd-catch-all-config-structured
service google-fluentd start
service google-fluentd restart

# Set environment variables
HOME=/root
GOPATH=${HOME}/go
GO111MODULE=on
PATH=${PATH}:/usr/local/go/bin:${GOPATH}/bin
export HOME GOPATH GO111MODULE PATH

# mkfs and mount local SSD on /var/scratch
mkfs -t ext4 -F /dev/disk/by-id/google-local-ssd-0
mkdir -p /var/scratch
mount -t ext4 /dev/disk/by-id/google-local-ssd-0 /var/scratch
chmod 1777 /var/scratch

# Run test
mkdir -p ${GOPATH}/src/github.com/cybozu-go
cd ${GOPATH}/src/github.com/cybozu-go
git clone https://github.com/cybozu-go/neco
cd neco/dctest
make setup placemat test SUITE=./bootstrap
`

	_, err = tmp.Write([]byte(script))
	if err != nil {
		return "", err
	}
	return tmp.Name(), nil
}

func deleteInstances(
	ctx context.Context,
	projectID string,
	zone string,
	doForce bool,
) error {
	instances, err := fetchInstanceNamesWithFilter(
		ctx,
		projectID,
		zone,
		excludeSkipAutoDeleteFilter,
	)
	if err != nil {
		return err
	}

	e := well.NewEnvironment(ctx)
	for n := range instances {
		n := n
		e.Go(func(ctx context.Context) error {
			cmd := []string{
				"gcloud", "compute", "instances", n,
				"--project", projectID,
				"--zone", zone,
			}
			c := well.CommandContext(ctx, cmd[0], cmd[1:]...)
			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Run()
		})
	}
	well.Stop()
	return well.Wait()
}
