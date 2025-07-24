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

func TestNodesAttrName(t *testing.T) {
	input := MakeNodes(pods...)

	want := MakeNodes(
		yaml.NewStringRNode("app1"),
		yaml.NewStringRNode("app2"),
		nil,
	)
	got := input.Attr("metadata").Attr("labels").Attr("app")

	cmpOpts := slices.Concat(commonCmpOpts, cmp.Options{
		cmpopts.IgnoreFields(Node{}, "parent", "lazySchema", "lookup"),
	})

	if diff := cmp.Diff(want, got, cmpOpts...); diff != "" {
		t.Fatalf("-want +got:\n%s", diff)
	}
}

func TestNodesSetValue(t *testing.T) {
	input := MakeNodes(pods...)

	//REVISIT: remove dependency on Attr()
	input.Attr("metadata").Attr("labels").Attr("app").SetValue(
		yaml.NewStringRNode("new-value").YNode(),
	)

	want := MakeNodes(
		yaml.NewStringRNode("new-value"),
		yaml.NewStringRNode("new-value"),
		yaml.NewStringRNode("new-value"),
	)
	got := input.Attr("metadata").Attr("labels").Attr("app")

	cmpOpts := slices.Concat(commonCmpOpts, cmp.Options{
		cmpopts.IgnoreFields(Node{}, "parent", "lazySchema", "lookup"),
	})

	if diff := cmp.Diff(want, got, cmpOpts...); diff != "" {
		t.Fatalf("-want +got:\n%s", diff)
	}
}
