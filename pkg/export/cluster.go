package export

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Mirantis/rekustomize/pkg/cleanup"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func Cluster(client Client, out kio.Writer, setPath bool) error {
	resources, err := client.ApiResources()
	if err != nil {
		return err
	}
	inputs := []kio.Reader{}
	for _, resource := range resources {
		// TODO: add another 'skip' layer before the Get call
		objects, err := client.Get(resource)
		if err != nil {
			return err
		}
		inputs = append(inputs, &kio.PackageBuffer{Nodes: objects})

		if setPath {
			for _, obj := range objects {
				SetObjectPath(obj, true)
			}
		}
	}

	pipeline := &kio.Pipeline{
		Inputs: inputs,
		Filters: []kio.Filter{
			cleanup.DefaultRules(),
			// REVISIT: FileSetter annotations are ignored - bug in kustomize?
			// 1) cleared if no annotation is set before the filter is applied
			// 2) reverted if path annotations was set before the filter is applied
			// &filters.FileSetter{FilenamePattern: "%n_%k.yaml"},
		},
		Outputs: []kio.Writer{out},
	}

	err = pipeline.Execute()
	return err
}

func SetObjectPath(obj *yaml.RNode, withNamespace bool) {
	annotations := map[string]string{}
	for k, v := range obj.GetAnnotations() {
		annotations[k] = v
	}
	path := fmt.Sprintf(
		"%s-%s.yaml",
		obj.GetName(),
		strings.ToLower(obj.GetKind()),
	)
	ns := obj.GetNamespace()
	if withNamespace && ns != "" {
		path = filepath.Join(ns, path)
	}
	annotations[kioutil.PathAnnotation] = path
	obj.SetAnnotations(annotations)
}
