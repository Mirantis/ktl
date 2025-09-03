package output

import (
	"errors"

	"github.com/Mirantis/ktl/pkg/apis"
	"github.com/Mirantis/ktl/pkg/types"
)

const (
	dirPerm = 0o700
)

var errMutuallyExclusive = errors.New("only one attribute allowed")

type Impl interface {
	Store(env *types.Env, resources *types.ClusterResources) error
}

func New(spec *apis.Output) (Impl, error) {
	if implSpec := spec.GetKustomize(); implSpec != nil {
		return newKustomizeOutput(implSpec)
	}

	if implSpec := spec.GetKustomizeComponents(); implSpec != nil {
		return newComponentsOutput(implSpec)
	}

	if implSpec := spec.GetHelmChart(); implSpec != nil {
		return newChartOutput(implSpec)
	}

	if implSpec := spec.GetCsv(); implSpec != nil {
		return newCSVOutput(implSpec)
	}

	if implSpec := spec.GetTable(); implSpec != nil {
		return newTableOutput(implSpec)
	}

	if implSpec := spec.GetCrdDescriptions(); implSpec != nil {
		return newCRDDescriptionsOutput(implSpec)
	}

	if implSpec := spec.GetKubectl(); implSpec != nil {
		return newKubectlOutput(implSpec)
	}

	if implSpec := spec.GetJson(); implSpec != nil {
		return newJSONOutput(implSpec)
	}

	return nil, errors.New("unsupported output")
}
