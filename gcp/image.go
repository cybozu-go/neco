package gcp

import (
	"context"
	"os"
)

// CreateVMXEnabledImage creates vmx-enabled image
func CreateVMXEnabledImage(ctx context.Context, cc *ComputeClient, cfgFile string) error {
	cc.DeleteInstance(ctx)

	err := cc.CreateVMXEnabledInstance(ctx)
	if err != nil {
		return err
	}

	err = cc.WaitInstance(ctx)
	if err != nil {
		return err
	}

	progFile, err := os.Executable()
	if err != nil {
		return err
	}

	err = cc.RunSetup(ctx, progFile, cfgFile)
	if err != nil {
		return err
	}

	err = cc.StopInstance(ctx)
	if err != nil {
		return err
	}

	cc.DeleteVMXEnabledImage(ctx)

	err = cc.CreateVMXEnabledImage(ctx)
	if err != nil {
		return err
	}

	err = cc.DeleteInstance(ctx)
	if err != nil {
		return err
	}

	return nil
}
