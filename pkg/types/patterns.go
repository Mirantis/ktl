package types

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/Mirantis/ktl/pkg/apis"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type StrList []string

func (l *StrList) UnmarshalYAML(node *yaml.Node) error {
	if node == nil {
		*l = nil

		return nil
	}

	if node.Kind == yaml.ScalarNode {
		return l.unmarshalScalar(node)
	}

	items := []string{}

	if err := node.Decode(&items); err != nil {
		return fmt.Errorf("invalid string list: %w", err)
	}

	*l = items

	return nil
}

func (l *StrList) unmarshalScalar(node *yaml.Node) error {
	var raw string
	if err := node.Decode(&raw); err != nil {
		return fmt.Errorf("invalid string list: %w", err)
	}

	buf := bytes.NewBufferString(raw)
	r := csv.NewReader(buf)

	rawParts, err := r.ReadAll()
	if err != nil {
		return fmt.Errorf("invalid string list: %w", err)
	}

	parts := slices.Concat(rawParts...)
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}

	*l = parts

	return nil
}

type Patterns StrList //nolint:recvcheck

func (p *Patterns) UnmarshalYAML(node *yaml.Node) error {
	var parts StrList
	if err := node.Decode(&parts); err != nil {
		return fmt.Errorf("invalid pattern: %w", err)
	}

	for _, pattern := range parts {
		if _, err := path.Match(pattern, ""); err != nil {
			return fmt.Errorf("invalid pattern: %w", err)
		}
	}

	*p = (Patterns)(parts)

	return nil
}

func (p Patterns) Match(name string) bool {
	for _, pattern := range p {
		match, err := path.Match(pattern, name)
		if err != nil {
			panic(err)
		}

		if match {
			return true
		}
	}

	return false
}

func NewPatternSelector(spec *apis.PatternSelector) (PatternSelector, error) {
	ps := PatternSelector{}

	for _, p := range spec.GetInclude() {
		if _, err := path.Match(p, ""); err != nil {
			return ps, fmt.Errorf("invalid pattern: %w", err)
		}
		ps.Include = append(ps.Include, p)
	}

	for _, p := range spec.GetExclude() {
		if _, err := path.Match(p, ""); err != nil {
			return ps, fmt.Errorf("invalid pattern: %w", err)
		}
		ps.Exclude = append(ps.Exclude, p)
	}

	return ps, nil
}

type PatternSelector struct {
	Include Patterns `json:"include" yaml:"include"`
	Exclude Patterns `json:"exclude" yaml:"exclude"`
}

func (sel *PatternSelector) Select(names []string) []string {
	if max(len(sel.Include), len(sel.Exclude)) == 0 {
		return names
	}

	selected := map[string]struct{}{}

	for _, name := range names {
		if len(sel.Include) == 0 || sel.Include.Match(name) {
			selected[name] = struct{}{}
		}
	}

	for _, name := range names {
		if sel.Exclude.Match(name) {
			delete(selected, name)
		}
	}

	result := []string{}

	for _, name := range names {
		if _, match := selected[name]; match {
			result = append(result, name)
		}
	}

	return result
}

func (sel *PatternSelector) UnmarshalYAML(node *yaml.Node) error {
	if node == nil {
		*sel = PatternSelector{}

		return nil
	}

	if node.Kind != yaml.MappingNode {
		return sel.unmarshalPattern(node)
	}

	type selStruct PatternSelector

	var plain selStruct

	if err := node.Decode(&plain); err != nil {
		return fmt.Errorf("invalid pattern selector: %w", err)
	}

	*sel = (PatternSelector)(plain)

	return nil
}

func (sel *PatternSelector) unmarshalPattern(node *yaml.Node) error {
	var patterns Patterns
	if err := node.Decode(&patterns); err != nil {
		return fmt.Errorf("invalid pattern selector: %w", err)
	}

	for _, part := range patterns {
		part, not := strings.CutPrefix(part, "-")
		if not {
			sel.Exclude = append(sel.Exclude, part)
		} else {
			sel.Include = append(sel.Include, part)
		}
	}

	return nil
}
