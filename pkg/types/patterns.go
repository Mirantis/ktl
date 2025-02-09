package types

import (
	"encoding/json"
	"path"
)

type Patterns []string

func (p *Patterns) UnmarshalJSON(data []byte) error {
	raw := []string{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	for _, pattern := range raw {
		if _, err := path.Match(pattern, ""); err != nil {
			return err
		}
	}
	*p = raw
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

type PatternSelector struct {
	Include Patterns `json:"include" yaml:"include"`
	Exclude Patterns `json:"exclude" yaml:"exclude"`
}

func (sel *PatternSelector) Select(names []string) []string {
	if 0 == max(len(sel.Include), len(sel.Exclude)) {
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
