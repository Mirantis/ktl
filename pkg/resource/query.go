package resource

import (
	"errors"
	"fmt"
	"iter"
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

func (p Query) IsWildcard() bool {
	return slices.ContainsFunc(p, yaml.IsWildcard)
}

func isListIndexLookup(p string) bool {
	if !yaml.IsListIndex(p) {
		return false
	}

	return strings.Contains(p, "=")
}

func (p Query) IsLookup() bool {
	return p.IsWildcard() || slices.ContainsFunc(p, isListIndexLookup)
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

type Queries[M any] struct {
	prefix  Query
	meta    M
	queries []*Queries[M]
}

func (qq *Queries[M]) lcp(query Query) int {
	maxlcp := min(len(query), len(qq.prefix))
	for lcp := range maxlcp {
		if qq.prefix[lcp] != query[lcp] {
			return lcp
		}
	}

	return maxlcp
}

func (qq *Queries[M]) Add(query Query, meta M) {
	lcp := qq.lcp(query)

	equals := lcp == max(len(query), len(qq.prefix))
	if equals || qq.prefix == nil {
		qq.prefix = query
		qq.meta = meta
		return
	}

	if lcp < len(qq.prefix) || len(qq.queries) == 0 {
		qq.split(lcp)
	}

	qq.addSub(query[lcp:], meta)
}

func (qq *Queries[M]) split(idx int) {
	var empty M
	left, right := qq.prefix[:idx], qq.prefix[idx:]

	subq := &Queries[M]{
		prefix:  right,
		meta:    qq.meta,
		queries: qq.queries,
	}

	qq.prefix = left
	qq.queries = []*Queries[M]{subq}
	qq.meta = empty

	return
}

func (qq *Queries[M]) addSub(query Query, meta M) {
	for _, subq := range qq.queries {
		if !subq.prefixMatch(query) {
			continue
		}

		subq.Add(query, meta)

		return
	}

	qq.queries = append(qq.queries, &Queries[M]{
		prefix: query,
		meta:   meta,
	})
}

func (qq *Queries[M]) prefixMatch(query Query) bool {
	if len(query) == 0 && len(qq.prefix) == 0 {
		return true
	}

	if len(query) == 0 || len(qq.prefix) == 0 {
		return false
	}

	return query[0] == qq.prefix[0]
}

func (qq *Queries[M]) Scan(node *yaml.RNode) iter.Seq2[M, *yaml.RNode] {
	return func(yield func(M, *yaml.RNode) bool) {
		for match := range qq.matchPrefix(node) {
			if len(qq.queries) == 0 {
				if !yield(qq.meta, match) {
					return
				}
				continue
			}

			for _, subQuery := range qq.queries {
				for meta, node := range subQuery.Scan(match) {
					if !yield(meta, node) {
						return
					}
				}
			}
		}
	}
}

func (qq *Queries[M]) matchPrefix(node *yaml.RNode) iter.Seq[*yaml.RNode] {
	return func(yield func(*yaml.RNode) bool) {
		if len(qq.prefix) == 0 {
			yield(node)
			return
		}

		//REVISIT: PathMatcher can only match a single [key=value]
		matcher := &yaml.PathMatcher{Path: qq.prefix}

		match, err := node.Pipe(matcher)
		if err != nil {
			panic(fmt.Errorf("broken matcher: %w", err))
		}

		matches, err := match.Elements()
		if err != nil {
			panic(fmt.Errorf("broken match: %w", err))
		}

		for _, matchNode := range matches {
			if !yield(matchNode) {
				return
			}
		}
	}
}

func NewQueries[M any](queries iter.Seq2[M, Query]) *Queries[M] {
	qq := &Queries[M]{}
	for meta, query := range queries {
		qq.Add(query, meta)
	}

	return qq
}
