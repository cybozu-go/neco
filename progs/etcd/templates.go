package etcd

import "text/template"

var confTmpl = template.Must(template.New("etcd.conf.yml").
	Parse(`# This is the configuration file for the etcd server.

# Human-readable name for this member.
name: 'boot-{{.LRN}}'

# Path to the data directory.
data-dir: '/var/lib/etcd'

# List of comma separated URLs to listen on for peer traffic.
listen-peer-urls: https://0.0.0.0:2380

# List of comma separated URLs to listen on for client traffic.
listen-client-urls: https://0.0.0.0:2379

# List of comma separated URLs to expose prometheus metrics.
listen-metrics-urls: http://0.0.0.0:2381

# List of this member's peer URLs to advertise to the rest of the cluster.
# The URLs needed to be a comma-separated list.
initial-advertise-peer-urls: {{.InitialAdvertisePeerURLs}}

# List of this member's client URLs to advertise to the public.
# The URLs needed to be a comma-separated list.
advertise-client-urls: {{.AdvertiseClientURLs}}

# Initial cluster configuration for bootstrapping.
initial-cluster: {{.InitialCluster}}

# Initial cluster token for the etcd cluster during bootstrap.
initial-cluster-token: 'boot-cluster'

# Initial cluster state ('new' or 'existing').
initial-cluster-state: '{{.InitialClusterState}}'

# Accept etcd V2 client requests
enable-v2: false

# Enable runtime profiling data via HTTP server
enable-pprof: true

# TLS certificates
client-transport-security:
  cert-file: /etc/neco/server.crt
  key-file: /etc/neco/server.key
  client-cert-auth: true
  trusted-ca-file: /etc/etcd/ca-client.crt

peer-transport-security:
  cert-file: /etc/etcd/peer.crt
  key-file: /etc/etcd/peer.key
  client-cert-auth: true
  trusted-ca-file: /etc/etcd/ca-peer.crt

# Specify 'stdout' or 'stderr' to skip journald logging even when running under systemd.
log-outputs: [stderr]

# auto compaction
auto-compaction-mode: periodic
auto-compaction-retention: "1"

# detect inconsistencies
# etcd 3.5.[0-2] has data inconsistency issue.
# https://groups.google.com/a/kubernetes.io/g/dev/c/B7gJs88XtQc/m/rSgNOzV2BwAJ
experimental-initial-corrupt-check: true

# Enabling data corruption detection
# https://etcd.io/docs/v3.5/op-guide/data_corruption/
experimental-compact-hash-check-enabled: true
# sigs.k8s.io/yaml only accepts integer values, nanoseconds.
# 10800000000000 nanoseconds = 3 hours
experimental-corrupt-check-time: 10800000000000
`))

var serviceTmpl = template.Must(template.New("etcd-container.service").
	Parse(`[Unit]
Description=Etcd container
Wants=network-online.target docker.service etcd-backup.timer
After=network-online.target docker.service
StartLimitIntervalSec=600s

[Service]
Type=simple
Restart=always
RestartSec=3s
OOMScoreAdjust=-1000
ExecStartPre=-/usr/bin/docker kill etcd
ExecStartPre=-/usr/bin/docker rm etcd
ExecStart=/usr/bin/docker run --name=etcd --rm \
  --network=host --uts=host \
  --log-driver=journald \
  --pull=never \
  --oom-kill-disable=true \
  --user={{ .UID }}:{{ .GID }} \
  --read-only \
  --volume=/etc/neco:/etc/neco:ro \
  --volume=/etc/ssl/certs:/etc/ssl/certs:ro \
  --volume=/etc/etcd:/etc/etcd:ro \
  --volume=/var/lib/etcd-container:/var/lib/etcd \
  {{ .Image }} --config-file={{ .ConfFile }}

[Install]
WantedBy=multi-user.target
`))
