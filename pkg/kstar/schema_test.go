package kstar

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"k8s.io/kube-openapi/pkg/validation/spec"
	"sigs.k8s.io/kustomize/kyaml/openapi"
)

func TestNodeSchemaResolve(t *testing.T) {
	const (
		metaRef      = `io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta`
		deployRef    = `io.k8s.api.apps.v1.Deployment`
		podSpecRef   = `io.k8s.api.core.v1.PodSpec`
		containerRef = `io.k8s.api.core.v1.Container`
	)
	idx := NewSchemaIndex(nil)
	root := &NodeSchema{
		idx: idx,
		ref: deployRef,
	}
	tests := []struct {
		name       string
		path       []string
		wantSchema spec.Schema
		wantRef    refName
		wantPath   fieldPath
	}{
		{
			name:       "root",
			path:       []string{},
			wantSchema: openapi.Schema().Definitions[deployRef],
			wantRef:    deployRef,
			wantPath:   nil,
		},
		{
			name:       "field",
			path:       []string{"metadata"},
			wantSchema: openapi.Schema().Definitions[metaRef],
			wantRef:    metaRef,
			wantPath:   nil,
		},
		{
			name:       "nested",
			path:       []string{"spec", "template", "spec", "containers"},
			wantSchema: openapi.Schema().Definitions[podSpecRef].Properties["containers"],
			wantRef:    podSpecRef,
			wantPath:   []string{"containers"},
		},
		{
			name:       "elements",
			path:       []string{"spec", "template", "spec", "containers", "[]"},
			wantSchema: openapi.Schema().Definitions[containerRef],
			wantRef:    containerRef,
			wantPath:   nil,
		},
		{
			name:     "unknown",
			path:     []string{"unknown"},
			wantRef:  deployRef,
			wantPath: []string{"unknown"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := root.Lookup(test.path...).Resolve()

			if diff := cmp.Diff(&test.wantSchema, got.schema, commonCmpOpts...); diff != "" {
				t.Errorf("schema -want +got:\n%s", diff)
			}

			if diff := cmp.Diff(test.wantRef, got.ref); diff != "" {
				t.Errorf("ref -want +got:\n%s", diff)
			}

			if diff := cmp.Diff(test.wantPath, got.path); diff != "" {
				t.Errorf("path -want +got:\n%s", diff)
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
