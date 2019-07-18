package teleport

import "text/template"

var confTmpl = template.Must(template.New("teleport.yaml").
	Parse(`# General configurations.
teleport:
  data_dir: /var/lib/teleport
  auth_token: %AUTH_TOKEN%
  advertise_ip: {{ .AdvertiseIP }}
  auth_servers: %AUTH_SERVERS%
  log:
    output: stderr
    severity: INFO
  storage:
    type: dir

# Node service specific configurations.
ssh_service:
  enabled: yes
  listen_addr: 0.0.0.0:3022
`))

var serviceTmpl = template.Must(template.New("teleport-node.service").
	Parse(`[Unit]
Description=Teleport node
Wants=network-online.target
After=network-online.target
StartLimitIntervalSec=600s
ConditionPathExists={{ .TokenFile }}

[Service]
Slice=system.slice
Type=simple
# SSH processes should be maintained even if main process is killed
KillMode=process
Restart=on-failure
RestartSec=10s
ExecStartPre=/usr/bin/neco teleport config
ExecStart=/usr/local/bin/teleport start --roles=node

[Install]
WantedBy=multi-user.target
`))
