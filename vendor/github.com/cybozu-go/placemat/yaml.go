package placemat

import (
	"bufio"
	"errors"
	"io"

	k8sYaml "github.com/kubernetes/apimachinery/pkg/util/yaml"
	yaml "gopkg.in/yaml.v2"
)

type baseConfig struct {
	Kind string `yaml:"kind"`
}

// ReadYaml reads a yaml file and constructs Cluster
func ReadYaml(r *bufio.Reader) (*Cluster, error) {
	var cluster Cluster

	y := k8sYaml.NewYAMLReader(r)
	for {
		data, err := y.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		var c baseConfig
		err = yaml.Unmarshal(data, &c)
		if err != nil {
			return nil, err
		}

		switch c.Kind {
		case "Network":
			spec := new(NetworkSpec)
			err = yaml.Unmarshal(data, spec)
			if err != nil {
				return nil, err
			}
			network, err := NewNetwork(spec)
			if err != nil {
				return nil, err
			}
			cluster.Networks = append(cluster.Networks, network)
		case "Image":
			spec := new(ImageSpec)
			err = yaml.Unmarshal(data, spec)
			if err != nil {
				return nil, err
			}
			image, err := NewImage(spec)
			if err != nil {
				return nil, err
			}
			cluster.Images = append(cluster.Images, image)
		case "DataFolder":
			spec := new(DataFolderSpec)
			err = yaml.Unmarshal(data, spec)
			if err != nil {
				return nil, err
			}
			folder, err := NewDataFolder(spec)
			if err != nil {
				return nil, err
			}
			cluster.DataFolders = append(cluster.DataFolders, folder)
		case "Node":
			spec := new(NodeSpec)
			err = yaml.Unmarshal(data, spec)
			if err != nil {
				return nil, err
			}
			node, err := NewNode(spec)
			if err != nil {
				return nil, err
			}
			cluster.Nodes = append(cluster.Nodes, node)
		case "Pod":
			spec := new(PodSpec)
			err = yaml.Unmarshal(data, spec)
			if err != nil {
				return nil, err
			}
			pod, err := NewPod(spec)
			if err != nil {
				return nil, err
			}
			cluster.Pods = append(cluster.Pods, pod)
		default:
			return nil, errors.New("unknown resource: " + c.Kind)
		}
	}
	return &cluster, nil
}
