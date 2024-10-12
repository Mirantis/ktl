package cleanup

import (
	"fmt"

	"sigs.k8s.io/kustomize/kyaml/resid"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var skipErr = fmt.Errorf("SKIP")

type skipRule []resid.ResId

func (r skipRule) Apply(rn *yaml.RNode) error {
	for i := range r {
		id := resid.FromRNode(rn)
		if id.IsSelectedBy(r[i]) {
			return skipErr
		}
	}
	return nil
}
