package filters

import (
	"fmt"
	"strings"

	"github.com/Mirantis/rekustomize/pkg/types"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func ClearAll(p types.NodePath) (yaml.Filter, error) {
	path, cond, err := p.Normalize()
	if err != nil {
		return nil, fmt.Errorf("invalid path %s: %v", p, err)
	}
	root := &yaml.TeePiper{}
	pipe := &root.Filters
	var pg *yaml.PathGetter
	for i := range path {
		if yaml.IsWildcard(path[i]) {
			forEach := &ForEach{}
			*pipe = append(*pipe, forEach)
			pipe = &forEach.Filters
			pg = nil
			continue
		}

		if cond[i] != "" {
			field, value, err := yaml.SplitIndexNameValue(cond[i])
			if err != nil {
				return nil, fmt.Errorf("invalid path condition %q: %v", cond[i], err)
			}
			matcher := yaml.FieldMatcher{
				Name: field,
			}
			fm := yaml.FilterMatcher{Filters: yaml.YFilters{{Filter: matcher}}}
			switch {
			case strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]"):
				fallthrough
			case strings.HasPrefix(value, "{") && strings.HasSuffix(value, "}"):
				rnValue, err := yaml.Parse(value)
				if err != nil {
					return nil, fmt.Errorf("invalid path condition %q: %v", cond[i], err)
				}
				matcher.Value = rnValue
			default:
				matcher.StringValue = value
			}
			*pipe = append(*pipe, fm)
		}

		if pg == nil {
			pg = &yaml.PathGetter{}
			*pipe = append(*pipe, pg)
		}

		pg.Path = append(pg.Path, path[i])
	}

	if pg != nil && len(pg.Path) > 0 {
		field := pg.Path[len(pg.Path)-1]
		pg.Path = pg.Path[:len(pg.Path)-1]
		*pipe = append(*pipe, yaml.Clear(field))
	} else {
		*pipe = append(*pipe, &ValueSetter{})
	}

	return root, nil
}
