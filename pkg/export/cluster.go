package export

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Mirantis/rekustomize/pkg/cleanup"
	"github.com/Mirantis/rekustomize/pkg/filter"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func Cluster(client Client, nsFilter, nsResFilter, clusterResFilter, selectors []string, out kio.Writer, setPath bool) error {
	allClusterResources, err := client.ApiResources(false)
	if err != nil {
		return err
	}
	clusterResources, err := filter.SelectNames(allClusterResources, clusterResFilter)
	if err != nil {
		return err
	}

	allNamespacedResources, err := client.ApiResources(true)
	if err != nil {
		return err
	}
	namespacedResources, err := filter.SelectNames(allNamespacedResources, nsResFilter)
	if err != nil {
		return err
	}

	allNamespaces, err := client.Namespaces()
	if err != nil {
		return err
	}
	namespaces, err := filter.SelectNames(allNamespaces, nsFilter)
	if err != nil {
		return err
	}

	inputs := []kio.Reader{}
	for _, resource := range clusterResources {
		names := []string{}
		if resource == "namespaces" {
			names = namespaces
		}
		objects, err := client.Get(resource, "", selectors, names...)
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
	for _, namespace := range namespaces {
		for _, resource := range namespacedResources {
			objects, err := client.Get(resource, namespace, selectors)
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
