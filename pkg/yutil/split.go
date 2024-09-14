package yutil

import (
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// RNodeEntry represents RNode with a Path
type RNodeEntry struct {
	Path Path
	*yaml.RNode
}

// Split returns a sequence of path/value entries
func Split(rn *yaml.RNode) ([]RNodeEntry, error) {
	return split(nil, rn)
}

func split(path Path, rn *yaml.RNode) ([]RNodeEntry, error) {
	if rn.IsNilOrEmpty() {
		return []RNodeEntry{{path, rn}}, nil
	}

	switch rn.YNode().Kind {
	case yaml.SequenceNode:
		return splitSequence(path, rn)
	case yaml.MappingNode:
		return splitMap(path, rn)
	case yaml.DocumentNode:
		return splitMap(path, rn)
	case yaml.ScalarNode:
		return []RNodeEntry{{path, rn}}, nil
	default:
		panic("not implemented")
	}
}

func splitSequence(path Path, rn *yaml.RNode) ([]RNodeEntry, error) {
	panic("not implemented")
}

func splitMap(path Path, rn *yaml.RNode) ([]RNodeEntry, error) {
	entries := []RNodeEntry{}
	fields, err := rn.Fields()
	if err != nil {
		return nil, err
	}
	for _, field := range fields {
		mapNode := rn.Field(field)
		subPath := append(path, field)
		subEntries, err := split(subPath, mapNode.Value)
		if err != nil {
			return nil, err
		}
		entries = append(entries, subEntries...)
	}
	return entries, nil
}
