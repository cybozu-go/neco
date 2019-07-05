package setuphw

import "text/template"

var serviceTmpl = template.Must(template.New("setup-hw.service").
	Parse(`[Unit]
Description=Setup hardware container
StartLimitIntervalSec=600s

[Service]
Slice=machine.slice
Type=simple
KillMode=mixed
Restart=on-failure
RestartSec=10s
OOMScoreAdjust=-1000
ExecStart=/usr/bin/rkt run \
  --pull-policy=never \
  --insecure-options=all \
  --net=host --dns=host --hosts-entry=host --hostname=%H \
  --volume dev,kind=host,source=/dev --mount volume=dev,target=/dev \
  --volume sys,kind=host,source=/sys --mount volume=sys,target=/sys \
  --volume modules,kind=host,source=/lib/modules,readOnly=true --mount volume=modules,target=/lib/modules \
  --volume neco,kind=host,source=/etc/neco,readOnly=true --mount volume=neco,target=/etc/neco \
  {{.Image}} \
    --name setup-hw \
    --caps-retain=CAP_SYS_ADMIN,CAP_SYS_CHROOT,CAP_CHOWN,CAP_FOWNER,CAP_NET_ADMIN

[Install]
WantedBy=multi-user.target
`))
