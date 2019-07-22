package setup

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"text/template"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/hashicorp/vault/api"
)

var etcdBackupTmpl = template.Must(template.New("").Parse(`#!/bin/sh

set -ue

SNAPSHOT=snapshot-$(date '+%Y%m%d_%H%M%S')
ETCDCTL="env ETCDCTL_API=3 /usr/local/bin/etcdctl --cert={{ .CertFile }} --key={{ .KeyFile }}"

$ETCDCTL snapshot save /var/lib/etcd-backup/${SNAPSHOT}
tar --remove-files -cvzf /var/lib/etcd-backup/${SNAPSHOT}.tar.gz /var/lib/etcd-backup/${SNAPSHOT}
find /var/lib/etcd-backup/ -mtime +14 -exec rm -f {} \;
`))

const etcdBackupService = `[Unit]
Description=get snapshot of etcd

[Service]
Type=oneshot
ExecStart=/usr/local/bin/etcd-backup
`

const etcdBackupTimer = `[Unit]
Description=get snapshot of etcd
PartOf=etcd-container.service

[Timer]
OnCalendar=hourly
RandomizedDelaySec=1min
Persistent=true

[Install]
WantedBy=timers.target
`

func setupEtcdBackup(ctx context.Context, vc *api.Client) error {
	err := os.MkdirAll(neco.EtcdBackupDir, 0700)
	if err != nil {
		return err
	}
	err = os.Chown(neco.EtcdBackupDir, neco.EtcdUID, neco.EtcdGID)
	if err != nil {
		return err
	}
	secret, err := vc.Logical().Write(neco.CAEtcdClient+"/issue/system", map[string]interface{}{
		"common_name":          "backup",
		"exclude_cn_from_sans": true,
	})
	if err != nil {
		return err
	}
	err = dumpCertFiles(secret, "", neco.EtcdBackupCertFile, neco.EtcdBackupKeyFile)
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	err = etcdBackupTmpl.Execute(buf, struct {
		CertFile string
		KeyFile  string
	}{
		CertFile: neco.EtcdBackupCertFile,
		KeyFile:  neco.EtcdBackupKeyFile,
	})
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("/usr/local/bin/etcd-backup", buf.Bytes(), 0755)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(neco.ServiceFile("etcd-backup"), []byte(etcdBackupService), 0644)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(neco.TimerFile("etcd-backup"), []byte(etcdBackupTimer), 0644)
	if err != nil {
		return err
	}
	err = neco.StartTimer(ctx, "etcd-backup")
	if err != nil {
		return err
	}
	log.Info("etcd-backup: installed", nil)

	return nil
}
