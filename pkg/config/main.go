package config

import (
	"github.com/Mirantis/rekustomize/pkg/types"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
)

const DefaultFileName = "rekustomization.yaml"

type Rekustomization struct {
	Source    Source          `yaml:"source"`
	HelmChart types.HelmChart `yaml:"helmChart"`

	Filters []filters.KFilter `yaml:"filters"`
}
