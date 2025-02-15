package types

import (
	"strings"

	"sigs.k8s.io/kustomize/kyaml/utils"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var rfc6901replacer = strings.NewReplacer("~", "~0", "/", "~1")

// NodePath represents YAML node path
type NodePath []string

// String returns a text representation of the path
func (p NodePath) String() string {
	if len(p) == 0 {
		return ""
	}

	escaped := make([]string, len(p))
	for i, part := range p {
		if strings.Contains(part, ".") {
			part = "[" + strings.Trim(part, "[]") + "]"
		}
		escaped[i] = part
	}
	return strings.Join(escaped, ".")
}

func (p *NodePath) UnmarshalYAML(node *yaml.Node) error {
	if node == nil {
		*p = nil
		return nil
	}
	if node.Kind == yaml.ScalarNode {
		return p.unmarshalScalar(node)
	}
	return node.Decode((*[]string)(p))
}

func (p *NodePath) unmarshalScalar(node *yaml.Node) error {
	var path string
	if err := node.Decode(&path); err != nil {
		return err
	}
	*p = utils.SmarterPathSplitter(path, ".")
	return nil
}
