package kstar

import (
	"fmt"
	"slices"
	"strings"
	"unicode"

	"go.starlark.net/starlark"
	"k8s.io/kube-openapi/pkg/validation/spec"
	"sigs.k8s.io/kustomize/kyaml/openapi"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	iok8s                   = "io.k8s."
	schemaDefinitionsPrefix = "#/definitions/"
	NodeSchemaType          = `NodeSchema`
	SchemaIndexType         = `SchemaIndex`
	schemaLookupType        = `SchemaLookup`
)

var (
	_ starlark.Value    = new(NodeSchema)
	_ starlark.Callable = new(NodeSchema)
)

type NodeSchema struct {
	idx    *SchemaIndex
	parent *NodeSchema
	schema *spec.Schema
	ref    refName
	path   fieldPath
}

func (ns *NodeSchema) String() string {
	panic(errNotImplemented)
}

func (ns *NodeSchema) Type() string {
	return NodeSchemaType
}

func (ns *NodeSchema) Freeze() {
	//TODO: freeze node
}

func (ns *NodeSchema) Truth() starlark.Bool {
	return ns != nil && (ns.schema != nil || len(ns.ref) > 0)
}

func (ns *NodeSchema) Hash() (uint32, error) {
	panic(errNotImplemented)
}

func (ns *NodeSchema) Name() string {
	name := NodeSchemaType
	if ns == nil {
		return name
	}

	if ns.ref != "" {
		name += "." + ns.ref
	}

	if len(ns.path) > 0 {
		name += "." + ns.path.String()
	}

	return name
}

func (ns *NodeSchema) callArgs(_ *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var value starlark.Value
	err := starlark.UnpackPositionalArgs(ns.Name(), args, kwargs, 1, &value)
	if err != nil {
		return nil, err
	}

	ynode, err := FromStarlark(value)
	if err != nil {
		return nil, err
	}

	node := FromYNode(ynode)
	node.setSchema(ns)

	return node, nil
}

func (ns *NodeSchema) callKWArgs(_ *starlark.Thread, _ starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	rnode := yaml.NewMapRNode(nil)
	node := &MappingNode{
		schema: ns,
		ynode:  rnode.YNode(),
	}

	for _, pair := range kwargs {
		name, value := pair[0].(starlark.String), pair[1]
		ynode, err := FromStarlark(value)
		if err != nil {
			return nil, err
		}

		path := strings.Split(name.GoString(), "_")
		vnode, err := rnode.Pipe(yaml.LookupCreate(ynode.Kind, path...))
		if err != nil {
			return nil, fmt.Errorf(
				"unable to set %v for %s: %w",
				path,
				ns.Name(),
				err,
			)
		}
		*vnode.YNode() = *ynode
	}

	return node, nil
}

func (ns *NodeSchema) CallInternal(th *starlark.Thread, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(args) > 0 {
		return ns.callArgs(th, args, kwargs)
	}

	return ns.callKWArgs(th, args, kwargs)
}

func (ns *NodeSchema) Schema() *openapi.ResourceSchema {
	ns = ns.Resolve()

	if ns == nil || ns.schema == nil {
		return nil
	}

	return &openapi.ResourceSchema{Schema: ns.schema}
}

func (ns *NodeSchema) Resolve() *NodeSchema {
	if ns == nil || ns.idx == nil || ns.schema != nil {
		return ns
	}

	if ns.ref != "" {
		schema := ns.idx.global.Definitions[ns.ref]
		ns.schema = &schema
		return ns
	}

	parent := ns.parent.Resolve()
	if parent == nil || parent.schema == nil {
		return ns
	}

	key := ns.path[0]
	if key == openapi.Elements {
		ns.schema = parent.schema.Items.Schema
	} else {
		schema := parent.schema.Properties[key]
		ns.schema = &schema
	}

	if ns.schema == nil {
		return ns
	}

	ns.ref = parent.ref
	ns.path = slices.Concat(parent.path, ns.path)

	ref := ns.schema.Ref.String()
	if ref == "" {
		return ns
	}

	ns.schema, _ = openapi.Resolve(&ns.schema.Ref, ns.idx.global)
	ns.ref = strings.TrimPrefix(ref, schemaDefinitionsPrefix)
	ns.path = nil

	return ns
}

func (ns *NodeSchema) Field(name string) *NodeSchema {
	if ns == nil {
		return nil
	}

	return &NodeSchema{
		idx:    ns.idx,
		parent: ns,
		path:   fieldPath{name},
	}
}

func (ns *NodeSchema) Elements() *NodeSchema {
	return ns.Field(openapi.Elements)
}

func (ns *NodeSchema) Lookup(path ...string) *NodeSchema {
	node := ns

	for _, part := range path {
		if part == openapi.Elements {
			node = node.Elements()
		} else {
			node = node.Field(part)
		}
	}

	return node
}

type refName = string

type fieldPath []string

func (path fieldPath) String() string {
	return strings.Join(path, ".")
}

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
			ref = strings.TrimPrefix(ref, schemaDefinitionsPrefix)
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

var (
	_ starlark.Value    = new(SchemaIndex)
	_ starlark.Mapping  = new(SchemaIndex)
	_ starlark.HasAttrs = new(SchemaIndex)
)

type SchemaIndex struct {
	cachedPaths map[refLink][]fieldPath
	refFields   map[refName]refFields
	global      *spec.Schema
	aliases     map[string]*NodeSchema
}

func NewSchemaIndex(schema *spec.Schema) *SchemaIndex {
	if schema == nil {
		schema = openapi.Schema()
	}

	aliases := map[string]*NodeSchema{}
	idx := &SchemaIndex{
		cachedPaths: map[refLink][]fieldPath{},
		refFields:   map[refName]refFields{},
		global:      schema,
		aliases:     aliases,
	}

	for ref := range schema.Definitions {
		if !strings.HasPrefix(ref, iok8s) {
			continue
		}

		parts := strings.Split(ref, ".")
		name := parts[len(parts)-1]
		aliases[name] = &NodeSchema{
			idx: idx,
			ref: ref,
		}
	}

	for ref := range schema.Definitions {
		if strings.HasPrefix(ref, iok8s) {
			continue
		}

		parts := strings.Split(ref, ".")
		name := parts[len(parts)-1]

		ns, dup := aliases[name]
		if dup && strings.HasPrefix(ns.ref, iok8s) {
			continue
		}

		if dup {
			aliases[name] = nil
			continue
		}

		aliases[name] = &NodeSchema{
			idx: idx,
			ref: ref,
		}
	}

	return idx
}

func (idx *SchemaIndex) String() string {
	panic(errNotImplemented)
}

func (idx *SchemaIndex) Type() string {
	return SchemaIndexType
}

func (idx *SchemaIndex) Freeze() {
	//TODO: freeze node
}

func (idx *SchemaIndex) Truth() starlark.Bool {
	return idx != nil && idx.global != nil
}

func (idx *SchemaIndex) Hash() (uint32, error) {
	panic(errNotImplemented)
}

func (idx *SchemaIndex) Attr(name string) (starlark.Value, error) {
	sl := &schemaLookup{idx: idx}

	return sl.Attr(name)
}

func (idx *SchemaIndex) AttrNames() []string {
	return nil
}

func (idx *SchemaIndex) Get(key starlark.Value) (_ starlark.Value, found bool, _ error) {
	name, ok := key.(starlark.String)
	if !ok {
		return nil, false, nil
	}

	if len(name) == 0 {
		return nil, false, nil
	}

	ref := name.GoString()
	_, found = idx.global.Definitions[ref]
	if !found {
		return nil, false, nil
	}

	schema := &NodeSchema{
		idx: idx,
		ref: ref,
	}

	return schema, found, nil
}

func (idx *SchemaIndex) rel(from, to refName) []fieldPath {
	if from == to {
		return []fieldPath{{}}
	}

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

var (
	_ starlark.Value    = new(schemaLookup)
	_ starlark.HasAttrs = new(schemaLookup)
)

type schemaLookup struct {
	idx   *SchemaIndex
	parts []string
}

func (sl *schemaLookup) String() string {
	panic(errNotImplemented)
}

func (sl *schemaLookup) Type() string {
	return schemaLookupType
}

func (sl *schemaLookup) Freeze() {
	//TODO: freeze node
}

func (sl *schemaLookup) Truth() starlark.Bool {
	return sl != nil && sl.idx.Truth()
}

func (sl *schemaLookup) Hash() (uint32, error) {
	panic(errNotImplemented)
}

func (sl *schemaLookup) AttrNames() []string {
	return nil
}

func (sl *schemaLookup) Attr(name string) (starlark.Value, error) {
	parts := []string{name}
	if len(sl.parts) > 0 {
		parts = slices.Concat(sl.parts, parts)
	}

	if unicode.IsLower(rune(name[0])) {
		return &schemaLookup{
			idx:   sl.idx,
			parts: parts,
		}, nil
	}

	name = strings.Join(parts, ".")
	if schema, found := sl.idx.aliases[name]; found && schema != nil {
		return schema, nil
	}

	matches := []string{}
	for ref := range sl.idx.global.Definitions {
		if !strings.HasSuffix(ref, name) {
			continue
		}

		matches = append(matches, ref)
	}

	sl.idx.aliases[name] = nil

	switch len(matches) {
	case 0:

		return nil, nil
	case 1:
		schema := &NodeSchema{
			idx: sl.idx,
			ref: matches[0],
		}
		sl.idx.aliases[name] = schema

		return schema, nil
	default:
		return nil, fmt.Errorf(
			"%w: multiple matching schemas for %q: %v",
			errInvalid,
			name,
			matches,
		)
	}
}
