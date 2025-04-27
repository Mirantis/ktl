package output

import (
	"errors"

	"github.com/Mirantis/rekustomize/pkg/types"
)

const (
	dirPerm = 0o700
)

var errMutuallyExclusive = errors.New("only one attribute allowed")

type Impl interface {
	Store(env *types.Env, resources *types.ClusterResources) error
}
