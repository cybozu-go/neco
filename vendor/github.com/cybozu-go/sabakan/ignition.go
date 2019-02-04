package sabakan

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"text/template"

	ignition "github.com/coreos/ignition/config/v2_3"
	"github.com/vincent-petithory/dataurl"
)

// MaxIgnitions is a number of the ignitions to keep on etcd
const MaxIgnitions = 10

// IgnitionInfo represents information of an ignition template
type IgnitionInfo struct {
	ID       string            `json:"id"`
	Metadata map[string]string `json:"meta"`
}

// IgnitionParams represents parameters in ignition template
type IgnitionParams struct {
	MyURL    *url.URL
	Machine  *Machine
	Metadata map[string]string
}

// ValidateIgnitionTemplate validates if the tmpl is a template for a valid ignition.
// The method returns nil if valid template is given, otherwise returns an error.
// The method returns template by tmpl nil value of Machine.
func ValidateIgnitionTemplate(tmpl string, metadata map[string]string, ipam *IPAMConfig) error {
	mc := NewMachine(MachineSpec{
		Serial: "1234abcd",
		Rack:   1,
	})
	u, err := url.Parse("http://localhost:10080")
	if err != nil {
		return err
	}

	ipam.GenerateIP(mc)
	ign, err := RenderIgnition(tmpl, &IgnitionParams{Metadata: metadata, Machine: mc, MyURL: u})
	if err != nil {
		return err
	}
	_, rpt, err := ignition.Parse([]byte(ign))
	if err != nil {
		return err
	}
	if len(rpt.Entries) > 0 {
		return errors.New(rpt.String())
	}
	return nil
}

// RenderIgnition returns the rendered ignition from the template and a machine
func RenderIgnition(tmpl string, params *IgnitionParams) (string, error) {
	var raw interface{}
	err := json.Unmarshal([]byte(tmpl), &raw)
	if err != nil {
		return "", err
	}

	dst, err := renderLeaves("", raw, params)
	if err != nil {
		return "", err
	}
	escapeDataURL(dst)

	b, err := json.Marshal(dst)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func renderLeaves(name string, src interface{}, params *IgnitionParams) (interface{}, error) {
	var err error
	switch x := src.(type) {
	case map[string]interface{}:
		m := make(map[string]interface{})
		for k, v := range x {
			m[k], err = renderLeaves(name+"."+k, v, params)
			if err != nil {
				return nil, err
			}
		}
		return m, nil
	case []interface{}:
		s := make([]interface{}, len(x))
		for i, v := range x {
			s[i], err = renderLeaves(fmt.Sprintf("%s[%d]", name, i), v, params)
			if err != nil {
				return nil, err
			}
		}
		return s, err
	case string:
		r, err := renderString(name, x, params)
		if err != nil {
			return nil, err
		}
		return r, nil
	}
	return src, nil
}

func escapeDataURL(src interface{}) {
	var root map[string]interface{}
	var ok bool
	if root, ok = src.(map[string]interface{}); !ok {
		return
	}
	var storage map[string]interface{}
	if storage, ok = root["storage"].(map[string]interface{}); !ok {
		return
	}
	var files []interface{}
	if files, ok = storage["files"].([]interface{}); !ok {
		return
	}

	for _, elem := range files {
		var file map[string]interface{}
		if file, ok = elem.(map[string]interface{}); !ok {
			continue
		}
		var contents map[string]interface{}
		if contents, ok = file["contents"].(map[string]interface{}); !ok {
			continue
		}
		var source string
		if source, ok = contents["source"].(string); !ok {
			continue
		}
		contents["source"] = "data:," + dataurl.EscapeString(source)
	}
}

func renderString(name string, src string, params *IgnitionParams) (string, error) {
	getMyURL := func() string {
		return params.MyURL.String()
	}
	getMetadata := func(key string) string {
		return params.Metadata[key]
	}

	var dst bytes.Buffer
	t, err := template.New(name).
		Funcs(template.FuncMap{
			"MyURL":    getMyURL,
			"Metadata": getMetadata,
			"json": func(i interface{}) (string, error) {
				data, err := json.Marshal(i)
				if err != nil {
					return "", err
				}
				return string(data), nil
			},
		}).
		Parse(src)
	if err != nil {
		return "", err
	}
	err = t.Execute(&dst, params.Machine)
	if err != nil {
		return "", err
	}
	return dst.String(), nil
}
