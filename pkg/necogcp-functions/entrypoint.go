package necogcpfunctions

import (
	"context"
	"encoding/json"
	"fmt"

	"log"

	"cloud.google.com/go/pubsub"
	"github.com/cybozu-go/neco/gcp/functions"
	"github.com/kelseyhightower/envconfig"
)

const (
	deleteInstancesMode = "delete"
	createInstancesMode = "create"

	necoBranch     = "release"
	necoAppsBranch = "release"

	machineType = "n1-standard-32"

	skipAutoDeleteLabelKey      = "skip-auto-delete"
	excludeSkipAutoDeleteFilter = "labels." + skipAutoDeleteLabelKey + "!=true"

	imageURL = "https://console.cloud.google.com/compute/imagesDetail/projects/neco-dev/global/images/vmx-enabled"
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

	client, err := functions.NewComputeClient(ctx, e.ProjectID, e.Zone)
	if err != nil {
		log.Printf("error: %v", err)
		return err
	}
	runner := functions.NewRunner(client)

	switch b.Mode {
	case createInstancesMode:
		log.Printf("info: check if today is holiday before creating instances")
		today, err := getDateStrInJST()
		if err != nil {
			log.Printf("error: %v", err)
			return err
		}
		if isHoliday(today, jpHolidays) {
			log.Printf("info: today %s is holiday! skip creating dctest", today)
			return nil
		}
		log.Printf("info: create %d instance(s) for %s", b.InstancesNum, b.InstanceNamePrefix)

		builder, err := functions.NewNecoStartupScriptBuilder().
			WithFluentd().
			WithNeco(necoBranch).
			WithNecoApps(necoAppsBranch)
		if err != nil {
			log.Printf("error: %v", err)
			return err
		}
		return runner.CreateInstancesIfNotExist(
			ctx,
			b.InstanceNamePrefix,
			b.InstancesNum,
			e.ServiceAccountName,
			machineType,
			imageURL,
			builder.Build(),
		)
	case deleteInstancesMode:
		log.Printf("info: delete all instances in %s with force=%t", e.ProjectID, b.DoForceDelete)
		return runner.DeleteInstancesWithFilter(ctx, excludeSkipAutoDeleteFilter)
	default:
		err := fmt.Errorf("invalid mode was given: %s", b.Mode)
		log.Fatalf("error: %v", err)
		return err
	}
}
