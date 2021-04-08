package teleport

import "text/template"

var confTmpl = template.Must(template.New("teleport.yaml").
	Parse(`# General configurations.
teleport:
  data_dir: /var/lib/teleport
  auth_token: {{ .AuthToken }}
  advertise_ip: {{ .AdvertiseIP }}
  auth_servers: [teleport-auth.teleport.svc.cluster.local:3025]
  log:
    output: stderr
    severity: INFO
  storage:
    type: dir

# Node service specific configurations.
ssh_service:
  enabled: yes
  listen_addr: 0.0.0.0:3022
  pam:
    enabled: yes
    service_name: "teleport"
`))
