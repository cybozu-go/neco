package serf

import "text/template"

type tags struct {
	OsName     string `json:"os-name"`
	OsVersion  string `json:"os-version"`
	Serial     string `json:"serial"`
	BootServer string `json:"boot-server"`
}

type serfConfig struct {
	Tags              tags     `json:"tags"`
	Interface         string   `json:"interface"`
	ReconnectInterval string   `json:"reconnect_interval"`
	ReconnectTimeout  string   `json:"reconnect_timeout"`
	TombstoneTimeout  string   `json:"tombstone_timeout"`
	RetryJoin         []string `json:"retry_join"`
	RetryMaxAttempts  int      `json:"retry_max_attempts"`
	RetryInterval     string   `json:"retry_interval"`
	LogLevel          string   `json:"log_level"`
	Protocol          int      `json:"protocol"`
}

var serviceTmpl = template.Must(template.New("serf.service").
	Parse(`[Unit]
Description=Serf container
Wants=docker.service time-sync.target
After=docker.service time-sync.target
ConditionPathExists={{ .ConfFile }}
StartLimitIntervalSec=600s

[Service]
Type=simple
Restart=always
RestartSec=10s
StartLimitInterval=10m
ExecStartPre=-/usr/bin/docker kill serf
ExecStartPre=-/usr/bin/docker rm serf
ExecStart=/usr/bin/docker run --name=serf --rm \
  --network=host --uts=host \
  --log-driver=journald \
  --pull=never \
  --read-only \
  --volume=/etc/serf:/etc/serf:ro \
  {{ .Image }} agent -config-file {{ .ConfFile }}

[Install]
WantedBy=multi-user.target
`))

var serviceTmplRkt = template.Must(template.New("serf.service").
	Parse(`[Unit]
Description=Serf container on rkt
Wants=time-sync.target
After=time-sync.target
ConditionPathExists=/etc/serf/serf.json
StartLimitIntervalSec=600s

[Service]
Slice=machine.slice
Type=simple
KillMode=mixed
Restart=on-failure
RestartSec=10s
StartLimitInterval=10m
ExecStart=/usr/bin/rkt run \
  --pull-policy never --net=host \
  --volume conf,kind=host,source=/etc/serf \
  --mount volume=conf,target=/etc/serf \
  --hostname %H \
  {{ .Image }} \
    --name serf \
    --readonly-rootfs=true \
  -- \
    agent -config-file {{ .ConfFile }}

[Install]
WantedBy=multi-user.target
`))
