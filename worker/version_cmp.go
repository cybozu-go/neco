package worker

import (
	"context"

	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
)

func (o *operator) needContainerImageUpdate(ctx context.Context, name string) (bool, error) {
	tag, err := o.storage.GetContainerTag(ctx, o.mylrn, name)
	if err == storage.ErrNotFound {
		return true, nil
	}
	if err != nil {
		return false, err
	}

	img, err := neco.CurrentArtifacts.FindContainerImage(name)
	if err != nil {
		return false, err
	}

	return img.Tag != tag, nil
}

func (o *operator) needDebUpdate(ctx context.Context, name string) (bool, error) {
	ver, err := o.storage.GetDebVersion(ctx, o.mylrn, name)
	if err == storage.ErrNotFound {
		return true, nil
	}
	if err != nil {
		return false, err
	}

	deb, err := neco.CurrentArtifacts.FindDebianPackage(name)
	if err != nil {
		return false, err
	}
	return ver != deb.Release, nil

}
