package kubectl

import (
	"github.com/Mirantis/rekustomize/pkg/types"
	"golang.org/x/sync/errgroup"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/resid"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type Clusters struct {
	*types.ClusterIndex
	cmd *Cmd
}

//nolint:lll
func (clusters *Clusters) Resources(selectors []types.ResourceSelector, filters []kio.Filter) (*types.ClusterResources, error) {
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
