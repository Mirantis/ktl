package types

import (
	"sigs.k8s.io/kustomize/api/types"
)

const (
	ComponentKind     = types.ComponentKind
	KustomizationKind = types.KustomizationKind
)

type Selector = types.Selector

type Kustomization = types.Kustomization

type Patch = types.Patch

type HelmChart = types.HelmChart

type HelmGlobals = types.HelmGlobals
