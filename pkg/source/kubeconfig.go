package source

import (
	"fmt"
	"log/slog"
	"maps"
	"slices"

	"github.com/Mirantis/rekustomize/pkg/kubectl"
	"github.com/Mirantis/rekustomize/pkg/types"
	"golang.org/x/sync/errgroup"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/resid"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type Kubeconfig struct {
	*types.ClusterIndex
	cmd *kubectl.Cmd
}

func NewKubeconfig(cmd *kubectl.Cmd, selectors []types.ClusterSelector) (*Kubeconfig, error) {
	names, err := cmd.Clusters()
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	clusters := &Kubeconfig{
		cmd:          cmd,
		ClusterIndex: types.BuildClusterIndex(names, selectors),
	}

	return clusters, nil
}

//nolint:lll
func (clusters *Kubeconfig) Resources(selectors []types.ResourceSelector, filters []kio.Filter) (*types.ClusterResources, error) {
	buffers := map[types.ClusterID]*kio.PackageBuffer{}
	errs := &errgroup.Group{}

	for clusterID, cluster := range clusters.All() {
		buffer := &kio.PackageBuffer{}
		buffers[clusterID] = buffer

		errs.Go(func() error {
			exporter, err := newClusterExporter(
				clusters.cmd.Cluster(cluster.Name),
				cluster.Name,
			)
			if err != nil {
				return err
			}

			return exporter.resources(buffer, selectors, filters)
		})
	}

	if err := errs.Wait(); err != nil {
		return nil, err //nolint:wrapcheck
	}

	resources := map[resid.ResId]map[types.ClusterID]*yaml.RNode{}
	result := &types.ClusterResources{
		Clusters:  clusters.ClusterIndex,
		Resources: resources,
	}

	for clusterID, buffer := range buffers {
		for _, rnode := range buffer.Nodes {
			id := resid.FromRNode(rnode)

			byResID, found := resources[id]
			if !found {
				byResID = map[types.ClusterID]*yaml.RNode{}
				resources[id] = byResID
			}

			byResID[clusterID] = rnode
		}
	}

	return result, nil
}

type clusterExporter struct {
	cmd  *kubectl.Cmd
	name string

	clusterResources    []string
	namespacedResources []string
	namespaces          []string
}

func newClusterExporter(cmd *kubectl.Cmd, name string) (*clusterExporter, error) {
	clusterResources, err := cmd.APIResources(false)
	if err != nil {
		return nil, fmt.Errorf("unable to get API resources list: %w", err)
	}

	namespacedResources, err := cmd.APIResources(true)
	if err != nil {
		return nil, fmt.Errorf("unable to get API resources list: %w", err)
	}

	namespaces, err := cmd.Namespaces()
	if err != nil {
		return nil, fmt.Errorf("unable to get namespaces list: %w", err)
	}

	exporter := &clusterExporter{
		cmd:  cmd,
		name: name,

		namespaces:          namespaces,
		namespacedResources: namespacedResources,
		clusterResources:    clusterResources,
	}

	return exporter, nil
}

func (c *clusterExporter) resources(out kio.Writer, selectors []types.ResourceSelector, filters []kio.Filter) error {
	nodes := map[resid.ResId]*yaml.RNode{}

	for _, rule := range selectors {
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
		Inputs:  inputs,
		Filters: filters,
		Outputs: []kio.Writer{out},
	}

	err := pipeline.Execute()
	if err != nil {
		return fmt.Errorf("export pipeline failed: %w", err)
	}

	return nil
}

func (c *clusterExporter) export(rule types.ResourceSelector) (map[resid.ResId]*yaml.RNode, error) {
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
		batch, err := c.cmd.Get(resources, ns, rule.LabelSelectors)
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
	}

	return byResID, nil
}
