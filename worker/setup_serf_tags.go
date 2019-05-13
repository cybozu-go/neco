package worker

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	setup_serf_tags "github.com/cybozu-go/neco/progs/setup-serf-tags"
)

const (
	setupSerfTags = "setup-serf-tags"
	binPath       = "/usr/local/bin/" + setupSerfTags

	setupSerfTagsService = `
[Unit]
Description=Set serf tags
Requires=serf.service
After=serf.service
PartOf=setup-serf-tags.timer

[Service]
Type=oneshot
ExecStart=` + binPath

	setupSerfTagsTimer = `
[Unit]
Description=Set serf tags periodically
Wants=serf.service
After=serf.service

[Timer]
OnCalendar=*-*-* *:*:0/20

[Install]
WantedBy=multi-user.target
`
)

func (o *operator) UpdateSetupSerfTags(ctx context.Context, req *neco.UpdateRequest) error {
	err := os.MkdirAll(filepath.Dir(neco.ServiceFile(setupSerfTags)), 0755)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(neco.ServiceFile(setupSerfTags), []byte(setupSerfTagsService), 0644)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(neco.TimerFile(setupSerfTags), []byte(setupSerfTagsTimer), 0644)
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	err = setup_serf_tags.GenerateScript(buf, req.Version)
	if err != nil {
		return err
	}
	_, err = replaceFile(binPath, buf.Bytes(), 0755)
	if err != nil {
		return err
	}
	err = neco.StartTimer(ctx, setupSerfTags)
	if err != nil {
		return err
	}
	log.Info(setupSerfTags+": installed", nil)

	return nil
}
