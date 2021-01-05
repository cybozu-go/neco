package neco

import (
	"context"
)

// Bind represents a host bind mount rule.
type Bind struct {
	Name     string
	Source   string
	Dest     string
	ReadOnly bool
}

// ContainerRuntime defines a set of operations to run containers on boot servers.
type ContainerRuntime interface {

	// ImageFullName returns the fully-qualified container image name.
	// The result for private images may vary depending on whether the container runtime
	// can access private image repositories.
	ImageFullName(img ContainerImage) string

	// Pull pulls the image.
	Pull(ctx context.Context, img ContainerImage) error

	// Run runs a container for the given image in front.
	Run(ctx context.Context, img ContainerImage, binds []Bind, args []string) error

	// Exec executes the given command in a running container named `name`.
	// The returned error is the error returned by exec.Cmd.Run().
	// If `stdio` is true, the command uses os.Stdin,out,err for I/O.
	Exec(ctx context.Context, name string, stdio bool, command []string) error

	// IsRunning returns true if there is a running container for the image.
	IsRunning(img ContainerImage) (bool, error)
}

// GetContainerRuntime() returns the container runtime for the running server.
// proxy may be used for some container runtimes.
func GetContainerRuntime(proxy string) (ContainerRuntime, error) {
	return newDockerRuntime(proxy)
}
