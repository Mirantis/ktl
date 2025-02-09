package export

import (
	"fmt"
	"log/slog"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Mirantis/rekustomize/pkg/types"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/resid"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type Cluster struct {
	Client
	Name  string
	Rules []types.ExportRule

	clusterResources    []string
	namespacedResources []string
	namespaces          []string
}

func (c *Cluster) init() error {
	var err error
	if c.clusterResources, err = c.Client.ApiResources(false); err != nil {
		return err
	}
	if c.namespacedResources, err = c.Client.ApiResources(true); err != nil {
		return err
	}
	if c.namespaces, err = c.Client.Namespaces(); err != nil {
		return err
	}
	return nil
}

func (c *Cluster) Execute(out kio.Writer, filters ...kio.Filter) error {
	if err := c.init(); err != nil {
		return err
	}

	nodes := map[resid.ResId]*yaml.RNode{}
	for _, rule := range c.Rules {
		batch, err := c.export(rule)
		if err != nil {
			return err
		}
		maps.Insert(nodes, maps.All(batch))
	}

	inputs := []kio.Reader{&kio.PackageBuffer{
		Nodes: slices.Collect(maps.Values(nodes)),
	}}
	pipeline := &kio.Pipeline{
		Inputs: inputs,
		// REVISIT: FileSetter annotations are ignored - bug in kustomize?
		// 1) cleared if no annotation is set before the filter is applied
		// 2) reverted if path annotations was set before the filter is applied
		// &filters.FileSetter{FilenamePattern: "%n_%k.yaml"},
		Filters: filters,
		Outputs: []kio.Writer{out},
	}

	return pipeline.Execute()
}

func (c *Cluster) export(rule types.ExportRule) (map[resid.ResId]*yaml.RNode, error) {
	slog.Info("exporting", "rule", rule)
	namespaces := slices.Clone(c.namespaces)
	resources := slices.Clone(c.namespacedResources)
	if 0 == max(len(rule.Namespaces.Include), len(rule.Namespaces.Exclude)) {
		namespaces = []string{""}
		resources = append(resources, c.clusterResources...)
	}
	namespaces = rule.Namespaces.Select(namespaces)
	resources = rule.Resources.Select(resources)

	nodes := []*yaml.RNode{}
	for _, ns := range namespaces {
		batch, err := c.Client.GetAll(ns, rule.LabelSelectors, resources...)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, batch...)
	}
	byResId := map[resid.ResId]*yaml.RNode{}
	for _, rn := range nodes {
		id := resid.FromRNode(rn)
		if 0 == len(rule.Names.Select([]string{id.Name})) {
			continue
		}
		byResId[id] = rn
		SetObjectPath(rn, true)
	}
	return byResId, nil
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
