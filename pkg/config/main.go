package config

import (
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
)

const DefaultFileName = "rekustomization.yaml"

type Rekustomization struct {
	Source Source `yaml:"source"`
	Output Output `yaml:"output"`

	Filters []filters.KFilter `yaml:"filters"`
}
