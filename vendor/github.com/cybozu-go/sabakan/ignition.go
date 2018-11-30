package sabakan

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"text/template"

	ignition "github.com/coreos/ignition/config/v2_2"
	"github.com/vincent-petithory/dataurl"
	yaml "gopkg.in/yaml.v2"
)

// MaxIgnitions is a number of the ignitions to keep on etcd
const MaxIgnitions = 10

// IgnitionInfo represents information of an ignition template
type IgnitionInfo struct {
	ID       string            `json:"id"`
	Metadata map[string]string `json:"meta"`
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
	ign, err := RenderIgnition(tmpl, metadata, mc, u)

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
func RenderIgnition(tmpl string, metadata map[string]string, m *Machine, myURL *url.URL) (string, error) {
	getMyURL := func() string {
		return myURL.String()
	}
	getMetadata := func(key string) string {
		return metadata[key]
	}
	t, err := template.New("ignition").
		Funcs(template.FuncMap{"MyURL": getMyURL, "Metadata": getMetadata}).
		Parse(tmpl)
	if err != nil {
		return "", err
	}
	buf := new(bytes.Buffer)
	err = t.Execute(buf, m.Spec)
	if err != nil {
		return "", err
	}

	var ign ignitionMap
	err = yaml.Unmarshal(buf.Bytes(), &ign)
	if err != nil {
		return "", err
	}
	ign.escapeDataURL()

	dataOut, err := json.Marshal(&ign)
	if err != nil {
		return "", err
	}

	return string(dataOut), nil
}

type ignitionMap struct {
	fields map[string]interface{}
}

func (i *ignitionMap) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var ign interface{}
	err := unmarshal(&ign)
	if err != nil {
		return err
	}
	ign = normalizeMap(ign)
	ignMap, ok := ign.(map[string]interface{})
	if !ok {
		return errors.New("invalid ignition, failed to convert")
	}
	i.fields = ignMap
	return nil
}

func (i *ignitionMap) MarshalJSON() ([]byte, error) {
	return json.Marshal(&i.fields)
}

func (i *ignitionMap) escapeDataURL() {
	if storageMap, ok := i.fields["storage"].(map[string]interface{}); ok {
		if files, ok := storageMap["files"].([]interface{}); ok {
			for _, elem := range files {
				if f, ok := elem.(map[string]interface{}); ok {
					if contents, ok := f["contents"].(map[string]interface{}); ok {
						if source, ok := contents["source"].(string); ok {
							contents["source"] = fmt.Sprintf("data:,%s", dataurl.EscapeString(source))
						}
					}
				}
			}
		}
	}
}

func normalizeMap(input interface{}) interface{} {
	switch x := input.(type) {
	case map[interface{}]interface{}:
		newMap := map[string]interface{}{}
		for k, v := range x {
			newMap[k.(string)] = normalizeMap(v)
		}
		return newMap
	case []interface{}:
		for i, v := range x {
			x[i] = normalizeMap(v)
		}
	}
	return input
}
