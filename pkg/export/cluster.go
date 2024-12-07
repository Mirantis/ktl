package export

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Mirantis/rekustomize/pkg/filter"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type Cluster struct {
	Client
	Name             string
	NsFilter         []string
	NsResFilter      []string
	ClusterResFilter []string
	Selectors        []string
}

func (c *Cluster) Execute(out kio.Writer, filters ...kio.Filter) error {
	allClusterResources, err := c.Client.ApiResources(false)
	if err != nil {
		return err
	}
	clusterResources, err := filter.SelectNames(allClusterResources, c.ClusterResFilter)
	if err != nil {
		return err
	}

	allNamespacedResources, err := c.Client.ApiResources(true)
	if err != nil {
		return err
	}
	namespacedResources, err := filter.SelectNames(allNamespacedResources, c.NsResFilter)
	if err != nil {
		return err
	}

	allNamespaces, err := c.Client.Namespaces()
	if err != nil {
		return err
	}
	namespaces, err := filter.SelectNames(allNamespaces, c.NsFilter)
	if err != nil {
		return err
	}
	slog.Info(
		"export",
		"cluster", c.Name,
		"namespaces", strings.Join(namespaces, ","),
		"namespaced-resources", strings.Join(namespacedResources, ","),
		"cluster-resources", strings.Join(clusterResources, ","),
	)

	inputs := []kio.Reader{}
	if nsidx := slices.Index(clusterResources, "namespaces"); nsidx >= 0 {
		clusterResources = slices.Delete(clusterResources, nsidx, nsidx+1)
		objects, err := c.Client.Get("namespaces", "", c.Selectors, namespaces...)
		if err != nil {
			return err
		}
		inputs = append(inputs, &kio.PackageBuffer{Nodes: objects})
		for _, obj := range objects {
			SetObjectPath(obj, true)
		}
	}
	for resources := range slices.Chunk(clusterResources, 30) {
		objects, err := c.Client.GetAll("", c.Selectors, resources...)
		if err != nil {
			return err
		}
		inputs = append(inputs, &kio.PackageBuffer{Nodes: objects})

		for _, obj := range objects {
			SetObjectPath(obj, true)
		}
	}
	for _, namespace := range namespaces {
		objects, err := c.Client.GetAll(namespace, c.Selectors, namespacedResources...)
		if err != nil {
			return err
		}
		inputs = append(inputs, &kio.PackageBuffer{Nodes: objects})

		for _, obj := range objects {
			SetObjectPath(obj, true)
		}
	}

	pipeline := &kio.Pipeline{
		Inputs: inputs,
		// REVISIT: FileSetter annotations are ignored - bug in kustomize?
		// 1) cleared if no annotation is set before the filter is applied
		// 2) reverted if path annotations was set before the filter is applied
		// &filters.FileSetter{FilenamePattern: "%n_%k.yaml"},
		Filters: filters,
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
