package setupserftags

import "text/template"

var scriptTmpl = template.Must(template.New("setup-serf-tags").Parse(`#!/bin/sh

# list failed unit names
# limit to 300 bytes because whole length of tags must be < 512 bytes.

# Ignore failure of backup-cke-etcd.service because it does not mean
# that the machine is not healthy.  Ideally, it should care only about
# the essential services related to the machine health.
systemd_units_failed="$(systemctl list-units --state=failed --no-legend --plain --full | grep -v backup-cke-etcd.service | cut -d' ' -f1  | tr '\n' ',' | head --bytes=300)"

/usr/local/bin/serf tags \
       -set uptime="$(uptime -s)" \
       -set version="{{ .Version }}" \
       -set systemd-units-failed="${systemd_units_failed}"
`))
