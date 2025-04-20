package runner

import (
	"path/filepath"

	"github.com/Mirantis/rekustomize/pkg/kubectl"
	"github.com/Mirantis/rekustomize/pkg/source"
	"github.com/Mirantis/rekustomize/pkg/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/kio"
)

type Source struct {
	WorkDir string
	Cmd     *kubectl.Cmd
	FileSys filesys.FileSystem

	Kustomization string                   `yaml:"kustomization"`
	KubeConfig    string                   `yaml:"kubeconfig"`
	Clusters      []types.ClusterSelector  `yaml:"clusters"`
	Resources     []types.ResourceSelector `yaml:"resources"`
}

func (src *Source) ClusterResources(filters []kio.Filter) (*types.ClusterResources, error) {
	if src.Kustomization != "" {
		return src.kustomization(filters)
	}

	return src.kubeconfig(filters)
}

func (src *Source) kustomization(filters []kio.Filter) (*types.ClusterResources, error) {
	path := src.Kustomization
	if !filepath.IsAbs(path) {
		path = filepath.Clean(filepath.Join(src.WorkDir, path))
	}

	kust, err := source.NewKustomize(
		src.Cmd,
		src.FileSys,
		path,
		src.Clusters,
	)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	return kust.Resources(src.Resources, filters) //nolint:wrapcheck
}

func (src *Source) kubeconfig(filters []kio.Filter) (*types.ClusterResources, error) {
	kubeconfig, err := source.NewKubeconfig(src.Cmd, src.Clusters)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	return kubeconfig.Resources(src.Resources, filters) //nolint:wrapcheck
}
