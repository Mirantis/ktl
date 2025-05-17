package output

import (
	"fmt"
	"slices"

	"github.com/Mirantis/rekustomize/pkg/resource"
	"github.com/Mirantis/rekustomize/pkg/types"
)

type KustomizeOutput struct{}

func (out *KustomizeOutput) Store(env *types.Env, resources *types.ClusterResources) error {
	kust := &types.Kustomization{}
	resourceStore := &resource.FileStore{
		FileSystem:    env.FileSys,
		NameGenerator: resource.FileName,
		PostProcessor: func(path string, body []byte) []byte {
			kust.Resources = append(kust.Resources, path)

			return body
		},
	}

	if err := resourceStore.WriteAll(resources.All()); err != nil {
		return fmt.Errorf("unable to store files: %w", err)
	}

	slices.Sort(kust.Resources)

	if err := resourceStore.WriteKustomization(kust); err != nil {
		return fmt.Errorf("unable to store kustomization: %w", err)
	}

	return nil
}
