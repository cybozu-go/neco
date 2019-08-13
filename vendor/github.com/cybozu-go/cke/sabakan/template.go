package sabakan

import (
	"errors"
	"strconv"

	"github.com/cybozu-go/cke"
)

// ValidateTemplate validates cluster template.
func ValidateTemplate(tmpl *cke.Cluster) error {
	if len(tmpl.Nodes) < 2 {
		return errors.New("template must contain at least two nodes")
	}

	roles := make(map[string]bool)
	var cpCount, ncpCount int
	for _, n := range tmpl.Nodes {
		if n.ControlPlane {
			cpCount++
			continue
		}

		ncpCount++
		roles[n.Labels[CKELabelRole]] = true
		if val, ok := n.Labels[CKELabelWeight]; ok {
			weight, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return errors.New("weight must be a float: " + val)
			}
			if weight <= 0 {
				return errors.New("weight must be positive: " + val)
			}
		}
	}

	if cpCount != 1 {
		return errors.New("template must contain only one control plane node")
	}
	if ncpCount >= 2 && ncpCount != len(roles) {
		return errors.New("non-control plane nodes must be associated with unique roles")
	}

	return tmpl.Validate(true)
}
