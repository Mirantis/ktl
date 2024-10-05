package cleanup

import (
	"regexp"
	"strings"

	"sigs.k8s.io/kustomize/kyaml/openapi"
	"sigs.k8s.io/kustomize/kyaml/yaml"
	"sigs.k8s.io/kustomize/kyaml/yaml/walk"
)

type schemaRule struct{}

func (schemaRule) Apply(rn *yaml.RNode) error {
	walker := walk.Walker{
		Visitor: &schemaVisitor{},
		Sources: walk.Sources{rn},
	}
	_, err := walker.Walk()
	return err
}

type schemaVisitor struct{}

var _ walk.Visitor = &schemaVisitor{}
var schemaDefaultsRegexp = regexp.MustCompile(`\. Defaults? to ("?[[:alnum:]]+"?)[ \.]`)

func (schemaVisitor) VisitScalar(nodes walk.Sources, schema *openapi.ResourceSchema) (*yaml.RNode, error) {
	// REVISIT: see https://github.com/itaysk/kubectl-neat/blob/master/pkg/defaults/defaults.go
	rn := nodes.Dest()
	if schema == nil || rn.IsNilOrEmpty() {
		return rn, nil
	}
	if strings.Contains(schema.Schema.Description, ". Read-only.") {
		return nil, nil
	}
	val := strings.TrimSpace(rn.MustString())
	defs := schemaDefaultsRegexp.FindStringSubmatch(schema.Schema.Description)
	if len(defs) != 2 {
		return rn, nil
	}
	if defs[1] == val || strings.Trim(defs[1], `"`) == val {
		return nil, nil
	}
	return rn, nil
}

func (schemaVisitor) VisitMap(nodes walk.Sources, _ *openapi.ResourceSchema) (*yaml.RNode, error) {
	return nodes.Dest(), nil
}

func (schemaVisitor) VisitList(nodes walk.Sources, _ *openapi.ResourceSchema, _ walk.ListKind) (*yaml.RNode, error) {
	return nodes.Dest(), nil
}
