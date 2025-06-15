package filters

import (
	"errors"

	"github.com/Mirantis/ktl/pkg/apis"
	kfilters "sigs.k8s.io/kustomize/kyaml/kio/filters"
)

func New(spec *apis.Filter) (kfilters.KFilter, error) {
	if impl := spec.GetSkip(); impl != nil {
		sf, err := newSkipFilter(impl)
		if err != nil {
			return kfilters.KFilter{}, err
		}

		return kfilters.KFilter{
			Filter: sf,
		}, nil
	}

	if impl := spec.GetStarlark(); impl != nil {
		sf, err := newStarlarkFilter(impl)
		if err != nil {
			return kfilters.KFilter{}, err
		}

		return kfilters.KFilter{
			Filter: sf,
		}, nil
	}

	return kfilters.KFilter{}, errors.New("unsupported filter")
}
