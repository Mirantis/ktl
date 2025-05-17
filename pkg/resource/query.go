package resource

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"sigs.k8s.io/kustomize/kyaml/utils"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var errNodePathInvalid = errors.New("invalid node path")

// Query represents YAML node path.
type Query []string //nolint:recvcheck

// String returns a text representation of the path.
func (p Query) String() string {
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

func (p Query) IsLookup() bool {
	for _, part := range p {
		if yaml.IsWildcard(part) {
			return true
		}

		if !yaml.IsListIndex(part) {
			continue
		}

		if strings.Contains(part, "=") {
			return true
		}
	}

	return false
}

func (p *Query) UnmarshalYAML(node *yaml.Node) error {
	if node == nil {
		*p = nil

		return nil
	}

	if node.Kind == yaml.ScalarNode {
		return p.unmarshalScalar(node)
	}

	err := node.Decode((*[]string)(p))
	if err != nil {
		return fmt.Errorf("invalid node path: %w", err)
	}

	return nil
}

func (p *Query) unmarshalScalar(node *yaml.Node) error {
	var path string
	if err := node.Decode(&path); err != nil {
		return fmt.Errorf("invalid node path: %w", err)
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
		key = pathPart[:splitIdx] //nolint:gocritic
		cond = pathPart[splitIdx:]
	}

	if suffix && strings.HasPrefix(cond, "[=") {
		cond = "[" + key + cond[1:]
		prefix = true
	}

	return key, cond, prefix
}

func mergeConditions(left, right string) string {
	if left == "" {
		return right
	}

	if right == "" {
		return left
	}

	return strings.TrimSuffix(left, "]") + "," + strings.TrimPrefix(right, "[")
}

func (p Query) Normalize() (Query, []string, error) {
	path := slices.Clone(p)
	conditions := make([]string, len(path))

	for idx := range p {
		key, cond, prefix := splitPathCondition(path[idx])
		if yaml.IsWildcard(key) && conditions[idx] != "" {
			return nil, nil, fmt.Errorf("%w: %s.%s", errNodePathInvalid, conditions[idx], key)
		}

		if prefix {
			path[idx] = key
			conditions[idx] = mergeConditions(conditions[idx], cond)

			continue
		}

		if idx == len(p)-1 {
			nextKey, _, err := yaml.SplitIndexNameValue(cond)
			if err != nil {
				return nil, nil, fmt.Errorf("%w: %v", errNodePathInvalid, err.Error())
			}

			path = append(path, nextKey)

			conditions = append(conditions, "") //nolint:makezero
		}

		path[idx] = key
		conditions[idx+1] = cond
	}

	return path, conditions, nil
}
