package resource_test

import (
	"testing"

	"github.com/Mirantis/ktl/pkg/resource"
	"github.com/Mirantis/ktl/pkg/types"
	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestBuilder(t *testing.T) {
	// REVISIT: split for more robust hit/miss testing
	items := []struct {
		path  types.NodePath
		kind  yaml.Kind
		value *yaml.Node
	}{
		{types.NodePath{"a"}, 0, yaml.NewMapRNode(nil).YNode()},
		{types.NodePath{"a", "b"}, 0, yaml.NewStringRNode("v1").YNode()},
		{types.NodePath{"x", "y", "z"}, yaml.SequenceNode, nil},
		{types.NodePath{"x", "y", "z", "[name=u]"}, yaml.MappingNode, nil},
		{types.NodePath{"x", "y", "z", "[name=u]", "v"}, 0, yaml.NewStringRNode("v2").YNode()},
		{types.NodePath{"x", "y", "z", "[name=w]"}, yaml.MappingNode, nil},
		{types.NodePath{"x", "y", "z", "[name=w]", "v"}, 0, yaml.NewStringRNode("v3").YNode()},
		{types.NodePath{"x", "l"}, 0, yaml.NewStringRNode("out-of-order").YNode()},
	}

	builder := resource.NewNodeBuilder(yaml.NewMapRNode(nil))

	for _, item := range items {
		var err error
		if item.value == nil {
			_, err = builder.Add(item.path, item.kind)
		} else {
			_, err = builder.Set(item.path, item.value)
		}

		if err != nil {
			t.Fatal(err)
		}
	}

	got := builder.RNode().MustString()
	want := `a:
  b: v1
x:
  y:
    z:
    - name: u
      v: v2
    - name: w
      v: v3
  l: out-of-order
`

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("+got -want:\n%s", diff)
	}

	if diff := cmp.Diff(5, builder.Hit); diff != "" {
		t.Errorf("hit (+got -want): %s", diff)
	}

	if diff := cmp.Diff(1, builder.Miss); diff != "" {
		t.Errorf("miss (+got -want): %s", diff)
	}
}
