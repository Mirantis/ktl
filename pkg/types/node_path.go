package types

import (
	"slices"
	"strings"

	"sigs.k8s.io/kustomize/kyaml/utils"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

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

func splitPathCondition(pathPart string) (key, cond string, prefix bool) {
	prefix = strings.HasPrefix(pathPart, "[")
	suffix := strings.HasSuffix(pathPart, "]")

	if prefix && suffix {
		return "*", pathPart, false
	}

	if !prefix && !suffix {
		return pathPart, "", true
	}

	if prefix {
		splitIdx := strings.LastIndex(pathPart, "]")
		cond = pathPart[:splitIdx+1]
		key = pathPart[splitIdx+1:]
	} else {
		splitIdx := strings.Index(pathPart, "[")
		key = pathPart[:splitIdx]
		cond = pathPart[splitIdx:]
	}

	if suffix && strings.HasPrefix(cond, "[=") {
		cond = "[" + key + cond[1:]
		prefix, suffix = true, false
	}

	return key, cond, prefix
}

func mergeConditions(left, right string) string {
	if "" == left {
		return right
	}
	if "" == right {
		return left
	}
	return strings.TrimSuffix(left, "]") + "," + strings.TrimPrefix(right, "[")
}

func (p NodePath) Normalize() (NodePath, []string, error) {
	path := slices.Clone(p)
	conditions := make([]string, len(path))
	for i := range p {
		key, cond, prefix := splitPathCondition(path[i])
		if prefix {
			path[i] = key
			conditions[i] = mergeConditions(conditions[i], cond)
			continue
		}

		if i == len(p)-1 {
			nextKey, _, err := yaml.SplitIndexNameValue(cond)
			if err != nil {
				return nil, nil, err
			}
			path = append(path, nextKey)
			conditions = append(conditions, "")
		}

		path[i] = key
		conditions[i+1] = cond
	}
	return path, conditions, nil
}
