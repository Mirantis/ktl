package kubectl

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

type Export struct {
	Client    Cmd
	Cluster   string
	Resources []types.ResourceSelector

	clusterResources    []string
	namespacedResources []string
	namespaces          []string
}

func (c *Export) init() error {
	var err error
	if c.clusterResources, err = c.Client.APIResources(false); err != nil {
		return fmt.Errorf("unable to get API resources list: %w", err)
	}

	if c.namespacedResources, err = c.Client.APIResources(true); err != nil {
		return fmt.Errorf("unable to get API resources list: %w", err)
	}

	if c.namespaces, err = c.Client.Namespaces(); err != nil {
		return fmt.Errorf("unable to get namespaces list: %w", err)
	}

	return nil
}

func (c *Export) Execute(out kio.Writer, filters ...kio.Filter) error {
	if err := c.init(); err != nil {
		return err
	}

	nodes := map[resid.ResId]*yaml.RNode{}

	for _, rule := range c.Resources {
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

	err := pipeline.Execute()
	if err != nil {
		return fmt.Errorf("export pipeline failed: %w", err)
	}

	return nil
}

func (c *Export) export(rule types.ResourceSelector) (map[resid.ResId]*yaml.RNode, error) {
	slog.Info("exporting", "rule", rule)

	namespaces := slices.Clone(c.namespaces)
	resources := slices.Clone(c.namespacedResources)

	if max(len(rule.Namespaces.Include), len(rule.Namespaces.Exclude)) == 0 {
		namespaces = []string{""}

		resources = append(resources, c.clusterResources...)
	}

	namespaces = rule.Namespaces.Select(namespaces)
	resources = rule.Resources.Select(resources)
	nodes := []*yaml.RNode{}

	for _, ns := range namespaces {
		batch, err := c.Client.GetAll(ns, rule.LabelSelectors, resources...)
		if err != nil {
			return nil, fmt.Errorf("unable to fetch resources: %w", err)
		}

		nodes = append(nodes, batch...)
	}

	byResID := map[resid.ResId]*yaml.RNode{}

	for _, resNode := range nodes {
		id := resid.FromRNode(resNode)
		if len(rule.Names.Select([]string{id.Name})) == 0 {
			continue
		}

		byResID[id] = resNode
		SetObjectPath(resNode, true)
	}

	return byResID, nil
}

func SetObjectPath(obj *yaml.RNode, withNamespace bool) {
	annotations := maps.Clone(obj.GetAnnotations())
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
	obj.SetAnnotations(annotations) //nolint:errcheck
}
