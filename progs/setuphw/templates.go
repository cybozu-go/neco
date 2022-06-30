package setuphw

import "text/template"

var serviceTmpl = template.Must(template.New("setup-hw.service").
	Parse(`[Unit]
Description=Setup hardware container
Wants=docker.service
After=docker.service
StartLimitIntervalSec=600s
ConditionPathExists=/etc/neco/bmc-address.json

[Service]
Type=simple
Restart=always
RestartSec=10s
OOMScoreAdjust=-1000
ExecStartPre=/bin/mkdir -p /var/lib/setup-hw
ExecStartPre=-/usr/bin/docker kill setup-hw
ExecStartPre=-/usr/bin/docker rm setup-hw
ExecStartPre=/usr/sbin/setup-setup-hw
ExecStart=/usr/bin/docker run --name=setup-hw --rm \
  --network=host --uts=host \
  --log-driver=journald \
  --pull=never \
  --oom-kill-disable=true \
  --privileged \
  --volume=/dev:/dev \
  --volume=/lib/modules:/lib/modules:ro \
  --volume=/etc/neco:/etc/neco:ro \
  --volume=/var/lib/setup-hw:/var/lib/setup-hw \
  {{ .Image }}

[Install]
WantedBy=multi-user.target
`))
