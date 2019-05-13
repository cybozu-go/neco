package setup_serf_tags

import (
	"io"
)

// GenerateScript generates script.
func GenerateScript(w io.Writer, version string) error {
	return scriptTmpl.Execute(w, struct {
		Version string
	}{
		Version: version,
	})
}
