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

var serviceTmplRkt = template.Must(template.New("setup-hw.service").
	Parse(`[Unit]
Description=Setup hardware container
StartLimitIntervalSec=600s
ConditionPathExists=/etc/neco/bmc-address.json

[Service]
Slice=machine.slice
Type=simple
KillMode=mixed
Restart=on-failure
RestartSec=10s
OOMScoreAdjust=-1000
ExecStartPre=/bin/mkdir -p /var/lib/setup-hw
ExecStart=/usr/bin/rkt run \
  --pull-policy=never \
  --insecure-options=all \
  --net=host --dns=host --hosts-entry=host --hostname=%H \
  --volume dev,kind=host,source=/dev --mount volume=dev,target=/dev \
  --volume sys,kind=host,source=/sys --mount volume=sys,target=/sys \
  --volume modules,kind=host,source=/lib/modules,readOnly=true --mount volume=modules,target=/lib/modules \
  --volume neco,kind=host,source=/etc/neco,readOnly=true --mount volume=neco,target=/etc/neco \
  --volume var,kind=host,source=/var/lib/setup-hw --mount volume=var,target=/var/lib/setup-hw \
  {{.Image}} \
    --name setup-hw \
    --caps-retain=CAP_SYS_ADMIN,CAP_SYS_CHROOT,CAP_CHOWN,CAP_FOWNER,CAP_NET_ADMIN

[Install]
WantedBy=multi-user.target
`))
