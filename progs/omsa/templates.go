package omsa

import "text/template"

var serviceTmpl = template.Must(template.New("setup-hw.service").
	Parse(`[Unit]
Description=Setup hardware container

[Service]
Slice=machine.slice
Type=simple
KillMode=mixed
Restart=always
RestartSec=10s
OOMScoreAdjust=-1000
ExecStart=/usr/bin/rkt run \
  --pull-policy=never \
  --insecure-options=all \
  --volume modules,kind=host,source=/lib/modules,readOnly=true \
  --mount volume=modules,target=/lib/modules \
  --volume dev,kind=host,source=/dev \
  --mount volume=dev,target=/dev \
  --volume neco,kind=host,source=/etc/neco,readOnly=true \
  --mount volume=neco,target=/etc/neco \
  {{.Image}} \
  --name setup-hw

[Install]
WantedBy=multi-user.target
`))
