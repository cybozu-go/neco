package systemdresolved

import "text/template"

var confTmpl = template.Must(template.New("neco.conf").
	Parse(`[Resolve]
DNS={{ .DNSAddress }}
`))
