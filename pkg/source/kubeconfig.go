package source

import (
	"fmt"
	"log/slog"
	"maps"
	"slices"

	"github.com/Mirantis/ktl/pkg/kubectl"
	"github.com/Mirantis/ktl/pkg/types"
	"golang.org/x/sync/errgroup"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/resid"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type Kubeconfig struct {
	Path      string                   `yaml:"kubeconfig"`
	Clusters  []types.ClusterSelector  `yaml:"clusters"`
	Resources []types.ResourceSelector `yaml:"resources"`
}

func (kcfg *Kubeconfig) UnmarshalYAML(node *yaml.Node) error {
	type kubeconfig Kubeconfig

	base := &kubeconfig{}
	if err := node.Decode(base); err != nil {
		return err //nolint:wrapcheck
	}

	kcfg.Path = base.Path
	kcfg.Clusters = base.Clusters
	kcfg.Resources = defaultResources(base.Resources)

	return nil
}

func (kcfg *Kubeconfig) Load(env *types.Env) (*State, error) {
	cmd := env.Cmd.SubCmd()
	if kcfg.Path != "" {
		cmd.Env = append(cmd.Env, "KUBECONFIG", kcfg.Path)
	}

	names, err := cmd.Clusters()
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	clusters := types.BuildClusterIndex(names, kcfg.Clusters)
	buffers := map[types.ClusterID]*kio.PackageBuffer{}
	errs := &errgroup.Group{}

	for clusterID, cluster := range clusters.All() {
		buffer := &kio.PackageBuffer{}
		buffers[clusterID] = buffer

		errs.Go(func() error {
			exporter, err := newClusterExporter(
				cmd.Cluster(cluster.Name),
				cluster.Name,
			)
			if err != nil {
				return err
			}

			nodes, err := exporter.resources(kcfg.Resources)
			if err != nil {
				return err
			}

			buffer.Nodes = nodes

			return nil
		})
	}

	if err := errs.Wait(); err != nil {
		return nil, err //nolint:wrapcheck
	}

	resources := map[types.ClusterID][]*yaml.RNode{}

	for clusterID, buffer := range buffers {
		resources[clusterID] = buffer.Nodes
	}

	state := &State{clusters, resources}

	return state, nil
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

func (c *clusterExporter) resources(selectors []types.ResourceSelector) ([]*yaml.RNode, error) {
	nodes := map[resid.ResId]*yaml.RNode{}

	for _, rule := range selectors {
		batch, err := c.export(rule)
		if err != nil {
			return nil, err
		}

		maps.Insert(nodes, maps.All(batch))
	}

	return slices.Collect(maps.Values(nodes)), nil
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
