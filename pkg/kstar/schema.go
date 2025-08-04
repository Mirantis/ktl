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

func newRefFields(schema *spec.Schema) refFields {
	refs := refFields{}
	if schema == nil {
		return refs
	}

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

	return refs
}

type refLink struct{ from, to refName }

type SchemaIndex struct {
	cachedPaths map[refLink][]fieldPath
	refFields   map[refName]refFields
	global      *spec.Schema
}

func NewSchemaIndex(schema *spec.Schema) *SchemaIndex {
	if schema == nil {
		schema = openapi.Schema()
	}

	return &SchemaIndex{
		cachedPaths: map[refLink][]fieldPath{},
		refFields:   map[refName]refFields{},
		global:      schema,
	}
}

func (idx *SchemaIndex) rel(from, to refName) []fieldPath {
	link := refLink{from: from, to: to}

	paths, cached := idx.cachedPaths[link]
	if cached {
		return paths
	}

	paths = idx.relDfs(from, to, map[refName]struct{}{})
	slices.SortFunc(paths, slices.Compare)
	idx.cachedPaths[link] = paths

	return paths
}

func (idx *SchemaIndex) relDfs(from, to refName, visited map[refName]struct{}) []fieldPath {
	var result []fieldPath
	visited[from] = struct{}{}

	fields, loaded := idx.refFields[from]
	if !loaded {
		schema := idx.schema(from)
		fields = newRefFields(schema)
		idx.refFields[from] = fields
	}

	for fieldRef, paths := range fields {
		if fieldRef == to {
			result = append(result, paths...)
			continue
		}

		if _, ok := visited[fieldRef]; ok {
			continue
		}

		for _, subPath := range idx.relDfs(fieldRef, to, visited) {
			for _, path := range paths {
				result = append(result, slices.Concat(path, subPath))
			}
		}
	}

	return result
}

func (idx *SchemaIndex) schema(ref refName) *spec.Schema {
	schema, found := idx.global.Definitions[ref]
	if !found {
		return nil
	}

	return &schema
}
