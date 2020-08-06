package functions

import (
	"context"
	"fmt"
	"log"

	"github.com/cybozu-go/well"
)

// Runner runs dctest environments on GCP instances
type Runner struct {
	compute *ComputeClient
}

// NewRunner creates Runner
func NewRunner(computeClient *ComputeClient) *Runner {
	return &Runner{compute: computeClient}
}

func (r Runner) makeInstanceName(prefix string, index int) string {
	return fmt.Sprintf("%s-%d", prefix, index)
}

// CreateInstancesIfNotExist lists instances not existing and create them
func (r Runner) CreateInstancesIfNotExist(
	ctx context.Context,
	instanceNamePrefix string,
	instancesNum int,
	serviceAccountName string,
	machineType string,
	imageURL string,
	startupScript string,
) error {
	set, err := r.compute.GetNameSet("")
	if err != nil {
		log.Printf("error: failed to get instances list because %q", err)
		return err
	}

	e := well.NewEnvironment(ctx)
	for i := 0; i < instancesNum; i++ {
		name := r.makeInstanceName(instanceNamePrefix, i)
		if _, ok := set[name]; ok {
			log.Printf("info: skip creating %s because it already exists", name)
			continue
		}

		e.Go(func(ctx context.Context) error {
			log.Printf("info: start creating %s", name)
			err := r.compute.Create(name, serviceAccountName, machineType, imageURL, startupScript)
			if err != nil {
				log.Printf("error: failed to create %s instance because %q", name, err)
				return err
			}
			log.Printf("info: create %s successfully", name)
			return nil
		})
	}
	well.Stop()
	return well.Wait()
}

// DeleteInstancesMatchingFilter deletes instances which match the given filter
func (r Runner) DeleteInstancesMatchingFilter(ctx context.Context, filter string) error {
	set, err := r.compute.GetNameSet(filter)
	if err != nil {
		log.Printf("error: failed to get instances list with %q because %q", filter, err)
		return err
	}

	e := well.NewEnvironment(ctx)
	for n := range set {
		name := n
		e.Go(func(ctx context.Context) error {
			log.Printf("info: start deleting %s", name)
			err := r.compute.Delete(name)
			if err != nil {
				log.Printf("error: failed to delete %s instance because %q", name, err)
				return err
			}
			log.Printf("info: delete %s successfully", name)
			return nil
		})
	}
	well.Stop()
	return well.Wait()
}

// MakeVMXEnabledImageURL returns vmx-enabled image URL in the project
func MakeVMXEnabledImageURL(projectID string) string {
	return "https://www.googleapis.com/compute/v1/projects/" + projectID + "/global/images/vmx-enabled"
}
