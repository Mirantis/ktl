package types

import (
	"sigs.k8s.io/kustomize/api/types"
)

type Selector = types.Selector
type Kustomization = types.Kustomization
type Patch = types.Patch

const (
	ComponentKind     = types.ComponentKind
	KustomizationKind = types.KustomizationKind
)
