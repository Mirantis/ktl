package output

import (
	"github.com/Mirantis/rekustomize/pkg/types"
)

const (
	dirPerm = 0o700
)

type Impl interface {
	Store(env *types.Env, resources *types.ClusterResources) error
}
