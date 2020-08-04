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
	compute "google.golang.org/api/compute/v1"
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

type gceClient struct {
	service         *compute.Service
	projectID       string
	zone            string
	projectEndpoint string
}

func newGCEClient(ctx context.Context, projectID, zone string) (*gceClient, error) {
	s, err := compute.NewService(ctx)
	if err != nil {
		return nil, err
	}

	return &gceClient{
		service:         s,
		projectID:       projectID,
		zone:            zone,
		projectEndpoint: "https://www.googleapis.com/compute/v1/projects/" + projectID,
	}, nil
}

func (c *gceClient) create(instanceName, serviceAccountName, machineType string) error {
	imageURL := "https://console.cloud.google.com/compute/imagesDetail/projects/neco-dev/global/images/vmx-enabled"

	instance := &compute.Instance{
		Name:        instanceName,
		MachineType: c.projectEndpoint + "/zones/" + c.zone + "/machineTypes/" + machineType,
		Disks: []*compute.AttachedDisk{
			{
				AutoDelete: true,
				Boot:       true,
				Type:       "PERSISTENT",
				InitializeParams: &compute.AttachedDiskInitializeParams{
					DiskName:    "root",
					SourceImage: imageURL,
				},
			},
		},
		NetworkInterfaces: []*compute.NetworkInterface{
			{
				AccessConfigs: []*compute.AccessConfig{
					{
						Type: "ONE_TO_ONE_NAT",
						Name: "External NAT",
					},
				},
				Network: c.projectEndpoint + "/global/networks/default",
			},
		},
		ServiceAccounts: []*compute.ServiceAccount{
			{
				Email: serviceAccountName,
				Scopes: []string{
					compute.DevstorageFullControlScope,
					compute.ComputeScope,
				},
			},
		},
	}

	_, err := c.service.Instances.Insert(c.projectID, c.zone, instance).Do()
	return err
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

	client, err := newGCEClient(ctx, e.ProjectID, e.Zone)
	if err != nil {
		log.Printf("error: %v", err)
		return err
	}

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
			client,
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
	client *gceClient,
) error {
	// instances, err := fetchInstanceNamesWithFilter(
	// 	ctx,
	// 	projectID,
	// 	zone,
	// 	"",
	// )
	// if err != nil {
	// 	return err
	// }

	tmpfilePath, err := createStartupScript(accountJSONPath)
	if err != nil {
		return err
	}
	defer os.Remove(tmpfilePath)

	e := well.NewEnvironment(ctx)
	for i := 0; i < instancesNum; i++ {
		name := makeInstanceName(instanceNamePrefix, i)
		// if _, ok := instances[name]; ok {
		// 	log.Printf("info: skip creating %s because it already exists", name)
		// 	continue
		// }

		e.Go(func(ctx context.Context) error {
			return client.create(name, serviceAccountName, machineType)
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
