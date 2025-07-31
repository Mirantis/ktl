package kstar

import (
	"slices"

	"sigs.k8s.io/kustomize/kyaml/openapi"
)

type NodeSchema struct {
	rs     *openapi.ResourceSchema
	parent *NodeSchema
	key    string
}

func (schema *NodeSchema) Root() *NodeSchema {
	root := schema

	for root != nil && root.parent != nil {
		root = root.parent
	}

	return root
}

func (schema *NodeSchema) Path() []string {
	path := []string{}

	for s := schema; s != nil && len(s.key) > 0; s = s.parent {
		path = append(path, s.key)
	}

	slices.Reverse(path)

	return path
}

func (schema *NodeSchema) Field(name string) *NodeSchema {
	if schema == nil {
		return nil
	}

	fieldSchema := schema.rs.Field(name)

	return &NodeSchema{
		rs: fieldSchema,

		parent: schema,
		key:    name,
	}
}

func (schema *NodeSchema) Elements() *NodeSchema {
	if schema == nil {
		return nil
	}

	fieldSchema := schema.rs.Elements()

	return &NodeSchema{
		rs:     fieldSchema,
		parent: schema,
		key:    openapi.Elements,
	}
}

func (schema *NodeSchema) Lookup(path ...string) *NodeSchema {
	result := schema

	for _, key := range path {
		if result == nil {
			return nil
		}

		if key == openapi.Elements {
			result = result.Elements()
		} else {
			result = result.Field(key)
		}
	}

	return result
}
