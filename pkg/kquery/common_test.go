package kquery

import (
	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var commonCmpOpts = []cmp.Option{
	cmp.AllowUnexported(Node{}),
	cmp.Transformer("RNodeAsYAML", func(rnode *yaml.RNode) string {
		return rnode.MustString()
	}),
}
