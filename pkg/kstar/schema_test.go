package kstar

import (
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"go.starlark.net/starlark"
	"k8s.io/kube-openapi/pkg/validation/spec"
	"sigs.k8s.io/kustomize/kyaml/openapi"
	"sigs.k8s.io/kustomize/kyaml/yaml"
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

func TestNodeSchemaCreate(t *testing.T) {
	const (
		metaRef   = `io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta`
		deployRef = `io.k8s.api.apps.v1.Deployment`
	)
	schemaIndex := NewSchemaIndex(nil)
	cmpOpts := slices.Concat(commonCmpOpts, cmp.Options{
		cmpopts.IgnoreFields(yaml.Node{}, "Line", "Style", "Column", "Tag"),
	})

	tests := []struct {
		name      string
		create    *NodeSchema
		script    string
		want      nodeValue
		wantErr   wantErr
		wantPanic wantPanic
	}{
		{
			name: "scalar",
			create: &NodeSchema{
				idx:  schemaIndex,
				ref:  metaRef,
				path: fieldPath{"name"},
			},
			script: `create("test")`,
			want: &ScalarNode{
				schema: &NodeSchema{
					idx:  schemaIndex,
					ref:  metaRef,
					path: fieldPath{"name"},
				},
				ynode: yaml.NewStringRNode(
					"test",
				).YNode(),
			},
		},
		{
			name: "sequence",
			create: &NodeSchema{
				idx:  schemaIndex,
				ref:  metaRef,
				path: fieldPath{"finalizers"},
			},
			script: `create(["a","b"])`,
			want: &SequenceNode{
				schema: &NodeSchema{
					idx:  schemaIndex,
					ref:  metaRef,
					path: fieldPath{"finalizers"},
				},
				ynode: yaml.NewListRNode(
					"a",
					"b",
				).YNode(),
			},
		},
		{
			name: "mapping",
			create: &NodeSchema{
				idx:  schemaIndex,
				ref:  metaRef,
				path: fieldPath{"labels"},
			},
			script: `create({"a":"b", "c":"d"})`,
			want: &MappingNode{
				schema: &NodeSchema{
					idx:  schemaIndex,
					ref:  metaRef,
					path: fieldPath{"labels"},
				},
				ynode: yaml.MustParse(strings.Join([]string{
					`a: b`,
					`c: d`,
				}, "\n")).YNode(),
			},
		},
		{
			name: "kwargs",
			create: &NodeSchema{
				idx: schemaIndex,
				ref: metaRef,
			},
			script: strings.Join([]string{
				`create(`,
				`  metadata_name="test",`,
				`  metadata_labels={`,
				`    "a": "b",`,
				`    "c": "d",`,
				`  },`,
				`)`,
			}, "\n"),
			want: &MappingNode{
				schema: &NodeSchema{
					idx: schemaIndex,
					ref: metaRef,
				},
				ynode: yaml.MustParse(strings.Join([]string{
					`metadata:`,
					`  name: test`,
					`  labels:`,
					`    a: b`,
					`    c: d`,
				}, "\n")).YNode(),
			},
		},
	}

	for _, test := range tests {
		runStarlarkTest(t, test.name,
			"result = "+test.script,
			StringDict{
				"create": test.create,
			},
			test.wantPanic, test.wantErr,
			func(t *testing.T, gotAll StringDict) {
				got, ok := gotAll["result"].(nodeValue)
				if !ok {
					t.Fatal("result is not a node")
				}

				if diff := cmp.Diff(test.want, got, cmpOpts...); diff != "" {
					t.Fatalf("-want +got:\n%s", diff)
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

func TestSchemaIndexLookup(t *testing.T) {
	const (
		deployRef  = `io.k8s.api.apps.v1.Deployment`
		customRef  = `com.mirantis.ktl.Deployment`
		customRef1 = `com.mirantis.ktl.v1.CustomDeployment`
		customRef2 = `com.mirantis.ktl.v2.CustomDeployment`
	)
	globalSchema := *openapi.Schema()
	globalSchema.Definitions = maps.Clone(globalSchema.Definitions)
	globalSchema.Definitions[customRef] = globalSchema.Definitions[deployRef]
	globalSchema.Definitions[customRef1] = globalSchema.Definitions[deployRef]
	globalSchema.Definitions[customRef2] = globalSchema.Definitions[deployRef]

	schemaIndex := NewSchemaIndex(&globalSchema)
	cmpOpts := slices.Concat(commonCmpOpts, cmp.Options{
		cmpopts.IgnoreFields(yaml.Node{}, "Line", "Style", "Column", "Tag"),
	})

	tests := []struct {
		name      string
		script    string
		want      starlark.Value
		wantErr   wantErr
		wantPanic wantPanic
	}{
		{
			name:   "exact-match",
			script: `schema["` + deployRef + `"]`,
			want: &NodeSchema{
				idx: schemaIndex,
				ref: deployRef,
			},
		},
		{
			name:    "exact-match",
			script:  `schema["no-such-schema"]`,
			wantErr: true,
		},
		{
			name:   "alias-match",
			script: `schema.Deployment`,
			want: &NodeSchema{
				idx: schemaIndex,
				ref: deployRef,
			},
		},
		{
			name:    "alias-no-match",
			script:  `schema.NoSuchSchema`,
			wantErr: true,
		},
		{
			name:   "partial-match",
			script: `schema.v1.Deployment`,
			want: &NodeSchema{
				idx: schemaIndex,
				ref: deployRef,
			},
		},
		{
			name:   "custom-match",
			script: `schema.ktl.Deployment`,
			want: &NodeSchema{
				idx: schemaIndex,
				ref: customRef,
			},
		},
		{
			name:   "custom-match-v1",
			script: `schema.v1.CustomDeployment`,
			want: &NodeSchema{
				idx: schemaIndex,
				ref: customRef1,
			},
		},
		{
			name:   "custom-match-v2",
			script: `schema.v2.CustomDeployment`,
			want: &NodeSchema{
				idx: schemaIndex,
				ref: customRef2,
			},
		},
		{
			name:   "long-match",
			script: `schema.` + deployRef,
			want: &NodeSchema{
				idx: schemaIndex,
				ref: deployRef,
			},
		},
		{
			name:    "duplicate-match",
			script:  `schema.CustomDeployment`,
			wantErr: true,
		},
		{
			name:    "lookup-no-match",
			script:  `schema.ktl.NoSuchSchema`,
			wantErr: true,
		},
		{
			name:    "invalid-key",
			script:  `schema[0]`,
			wantErr: true,
		},
		{
			name:    "invalid-key",
			script:  `schema[0]`,
			wantErr: true,
		},
		{
			name:    "empty-key",
			script:  `schema[""]`,
			wantErr: true,
		},
	}

	for _, test := range tests {
		runStarlarkTest(t, test.name,
			"result = "+test.script,
			StringDict{
				"schema": schemaIndex,
			},
			test.wantPanic, test.wantErr,
			func(t *testing.T, gotAll StringDict) {
				got := gotAll["result"]

				if diff := cmp.Diff(test.want, got, cmpOpts...); diff != "" {
					t.Fatalf("-want +got:\n%s", diff)
				}
			})
	}
}
