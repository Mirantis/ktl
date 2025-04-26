package output

import (
	"fmt"
	"path/filepath"
	"slices"

	"github.com/Mirantis/rekustomize/pkg/resource"
	"github.com/Mirantis/rekustomize/pkg/types"
)

type KustomizeOutput struct{}

func (out *KustomizeOutput) Store(env *types.Env, resources *types.ClusterResources) error {
	kust := &types.Kustomization{}
	resourceStore := &resource.FileStore{
		Dir:           env.WorkDir,
		FileSystem:    env.FileSys,
		NameGenerator: resource.FileName,
		PostProcessor: func(path string, body []byte) []byte {
			relPath, err := filepath.Rel(env.WorkDir, path)
			if err != nil {
				panic(err)
			}
			kust.Resources = append(kust.Resources, relPath)

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
