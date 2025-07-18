package output

import (
	"errors"

	"github.com/Mirantis/ktl/pkg/apis"
	"github.com/Mirantis/ktl/pkg/types"
	"golang.org/x/sync/errgroup"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// TODO: add tests
type kubectlOutput struct {
	spec *apis.KubectlOutput
}

func newKubectlOutput(spec *apis.KubectlOutput) (Impl, error) {
	return &kubectlOutput{spec}, nil
}

var errUnknownOutputCluster = errors.New("unable to infer output cluster from source")

func (out *kubectlOutput) Store(env *types.Env, resources *types.ClusterResources) error {
	singleCluster := out.spec.GetCluster()
	if len(resources.Clusters.IDs()) > 1 {
		singleCluster = ""
	}

	errs := &errgroup.Group{}

	for clusterID, cluster := range resources.Clusters.All() {
		clusterName := cluster.Name
		if singleCluster != "" {
			clusterName = singleCluster
		}

		if clusterName == "" {
			return errUnknownOutputCluster
		}

		rnodes := []*yaml.RNode{}
		for _, rnode := range resources.All(&clusterID) {
			rnodes = append(rnodes, rnode)
		}

		cmd := env.Cmd.Cluster(clusterName)
		errs.Go(func() error {
			return cmd.Apply(rnodes)
		})
	}

	return errs.Wait()
}
