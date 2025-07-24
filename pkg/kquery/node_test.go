package kquery

import (
	"bytes"
	_ "embed"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var (
	//go:embed testdata/pods.yaml
	podsYaml []byte
	pods, _  = (&kio.ByteReader{Reader: bytes.NewBuffer(podsYaml)}).Read()
)

func TestNodeSetAttrName(t *testing.T) {
	input := MakeNodes(pods...)

	want := MakeNodes(
		yaml.NewStringRNode("app1"),
		yaml.NewStringRNode("app2"),
		nil,
	)
	got := input.Attr("metadata").Attr("labels").Attr("app")

	cmpOpts = append(slices.Clone(cmpOpts),
		cmpopts.IgnoreFields(Node{}, "parent", "lazySchema"),
		cmpopts.IgnoreFields(Nodes{}, "parent", "lookup"),
	)

	if diff := cmp.Diff(want, got, cmpOpts...); diff != "" {
		t.Fatalf("-want +got:\n%s", diff)
	}
}

