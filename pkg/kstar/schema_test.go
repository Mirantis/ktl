package kstar

import (
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"k8s.io/kube-openapi/pkg/validation/spec"
	"sigs.k8s.io/kustomize/kyaml/openapi"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestSchemaPath(t *testing.T) {
	cmpOpts := slices.Concat(commonCmpOpts, cmp.Options{
		cmp.AllowUnexported(spec.Ref{}),
		cmpopts.IgnoreTypes(spec.Ref{}.Ref),
	})
	podSchema := &NodeSchema{rs: openapi.SchemaForResourceType(yaml.TypeMeta{
		Kind:       "Pod",
		APIVersion: "v1",
	})}

	tests := []struct {
		name      string
		input     *NodeSchema
		path      []string
		wantPanic wantPanic
	}{
		{
			name:  "metadata.name",
			input: podSchema,
			path:  []string{"metadata", "name"},
		},
		{
			name:  "container.name",
			input: podSchema,
			path:  []string{"spec", "containers", "[]", "name"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer test.wantPanic.recover(t)

			gotSchema := test.input.Lookup(test.path...)

			if test.wantPanic.check(t) {
				return
			}

			gotPath := gotSchema.Path()
			if diff := cmp.Diff(test.path, gotPath); diff != "" {
				t.Fatalf("path mismatch, -want +got:\n%s", diff)
			}

			wantSchema := test.input.rs.Lookup(test.path...).Schema
			if diff := cmp.Diff(wantSchema, gotSchema.rs.Schema, cmpOpts...); diff != "" {
				t.Fatalf("schema mismatch, -want +got:\n%s", diff)
			}
		})
	}
}
