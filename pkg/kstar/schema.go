package kstar

import (
	"slices"
	"strings"

	"k8s.io/kube-openapi/pkg/validation/spec"
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

type refName = string

type fieldPath []string

type refFields map[refName][]fieldPath

func (refs refFields) load(schema *spec.Schema) error {
	type qentry struct {
		schema *spec.Schema
		path   fieldPath
	}

	queue := []qentry{{schema: schema}}
	for len(queue) > 0 {
		curr := queue[len(queue)-1]
		queue = queue[:len(queue)-1]

		ref := curr.schema.Ref.String()
		if ref != "" {
			ref, _ = strings.CutPrefix(ref, "#/definitions/")
			refs[ref] = append(refs[ref], curr.path)
			continue
		}

		var iSchemas = []spec.Schema{}
		if curr.schema.Items != nil {
			iSchemas = slices.Clone(curr.schema.Items.Schemas)
			if curr.schema.Items.Schema != nil {
				iSchemas = append(iSchemas, *curr.schema.Items.Schema)
			}
		}

		for _, iSchema := range iSchemas {
			queue = append(queue, qentry{
				schema: &iSchema,
				path:   slices.Concat(curr.path, fieldPath{openapi.Elements}),
			})
		}

		for pName, pSchema := range curr.schema.Properties {
			queue = append(queue, qentry{
				schema: &pSchema,
				path:   slices.Concat(curr.path, fieldPath{pName}),
			})
		}
	}

	for _, paths := range refs {
		slices.SortFunc(paths, slices.Compare)
	}

	return nil
}
