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

func TestFieldIndexLoad(t *testing.T) {
	const metaRef = "io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta"
	want := refFields{
		"io.k8s.apimachinery.pkg.apis.meta.v1.ManagedFieldsEntry": {
			{"managedFields", "[]"},
		},
		"io.k8s.apimachinery.pkg.apis.meta.v1.OwnerReference": {
			{"ownerReferences", "[]"},
		},
		"io.k8s.apimachinery.pkg.apis.meta.v1.Time": {
			{"creationTimestamp"},
			{"deletionTimestamp"},
		},
	}

	schema := openapi.Schema().Definitions[metaRef]
	got := newRefFields(&schema)

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("-want +got:\n%s", diff)
	}
}

func TestSchemaIndexRel(t *testing.T) {
	tests := []struct {
		name string
		from refName
		to   refName
		want []fieldPath
	}{
		{
			name: "linked",
			from: `io.k8s.api.apps.v1.Deployment`,
			to:   `io.k8s.api.core.v1.EnvVar`,
			want: []fieldPath{
				{"spec", "template", "spec", "containers", "[]", "env", "[]"},
				{"spec", "template", "spec", "ephemeralContainers", "[]", "env", "[]"},
				{"spec", "template", "spec", "initContainers", "[]", "env", "[]"},
			},
		},
		{
			name: "not-linked",
			from: `io.k8s.api.core.v1.ConfigMap`,
			to:   `io.k8s.api.core.v1.EnvVar`,
			want: nil,
		},
		{
			name: "not-linked-cached",
			from: `io.k8s.api.core.v1.ConfigMap`,
			to:   `io.k8s.api.core.v1.EnvVar`,
			want: nil,
		},
		{
			name: "undef",
			from: `undef1`,
			to:   `undef2`,
			want: nil,
		},
	}

	idx := NewSchemaIndex(nil)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := idx.rel(test.from, test.to)

			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Fatalf("-want +got:\n%s", diff)
			}
		})
	}
}
