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
			fm := yaml.FilterMatcher{}
			switch {
			case value == "":
				fallthrough
			case strings.HasPrefix(value, "{"):
				fallthrough
			case strings.HasPrefix(value, "["):
				fm.Filters = append(
					fm.Filters,
					yaml.YFilter{Filter: yaml.Lookup(field)},
					yaml.YFilter{Filter: &ValueMatcher{Value: value}},
				)
			default:
				matcher := yaml.FieldMatcher{
					Name: field,
				}
				matcher.StringValue = value
				fm.Filters = append(fm.Filters, yaml.YFilter{Filter: matcher})
			}
			*pipe = append(*pipe, fm)
			pg = nil
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
