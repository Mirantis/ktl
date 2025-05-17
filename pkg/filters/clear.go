package filters

import (
	"fmt"
	"strings"

	"github.com/Mirantis/ktl/pkg/resource"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func makeFieldMatcher(field, value string) *yaml.FilterMatcher {
	filterMatcher := &yaml.FilterMatcher{}

	switch {
	case value == "":
		fallthrough
	case strings.HasPrefix(value, "{"):
		fallthrough
	case strings.HasPrefix(value, "["):
		filterMatcher.Filters = append(
			filterMatcher.Filters,
			yaml.YFilter{Filter: yaml.Lookup(field)},
			yaml.YFilter{Filter: &ValueMatcher{Value: value}},
		)
	default:
		matcher := yaml.FieldMatcher{
			Name: field,
		}
		matcher.StringValue = value
		filterMatcher.Filters = append(filterMatcher.Filters, yaml.YFilter{Filter: matcher})
	}

	return filterMatcher
}

func ClearAll(p resource.Query) (*yaml.TeePiper, error) {
	path, cond, err := p.Normalize()
	if err != nil {
		return nil, fmt.Errorf("invalid path %s: %w", p, err)
	}

	root := &yaml.TeePiper{}
	pipe := &root.Filters

	var pathGetter *yaml.PathGetter

	for idx := range path {
		if yaml.IsWildcard(path[idx]) {
			forEach := &ForEach{}
			*pipe = append(*pipe, forEach)
			pipe = &forEach.Filters
			pathGetter = nil

			continue
		}

		if cond[idx] != "" {
			field, value, err := yaml.SplitIndexNameValue(cond[idx])
			if err != nil {
				return nil, fmt.Errorf("invalid path condition %q: %w", cond[idx], err)
			}

			*pipe = append(*pipe, makeFieldMatcher(field, value))
			pathGetter = nil
		}

		if pathGetter == nil {
			pathGetter = &yaml.PathGetter{}
			*pipe = append(*pipe, pathGetter)
		}

		pathGetter.Path = append(pathGetter.Path, path[idx])
	}

	if pathGetter != nil && len(pathGetter.Path) > 0 {
		field := pathGetter.Path[len(pathGetter.Path)-1]
		pathGetter.Path = pathGetter.Path[:len(pathGetter.Path)-1]

		*pipe = append(*pipe, yaml.Clear(field))
	} else {
		*pipe = append(*pipe, &ValueSetter{})
	}

	return root, nil
}
