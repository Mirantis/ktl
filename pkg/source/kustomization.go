package source

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Mirantis/rekustomize/pkg/types"
	"golang.org/x/sync/errgroup"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type Kustomize struct {
	PathTemplate string                   `yaml:"kustomization"`
	Clusters     []types.ClusterSelector  `yaml:"clusters"`
	Resources    []types.ResourceSelector `yaml:"resources"`
}

type kustomizePkg struct {
	idx   *types.ClusterIndex
	paths map[types.ClusterID]string
}

var (
	errPlaceholderMissing   = errors.New("missing " + types.ClusterPlaceholder)
	errMultiplePlaceholders = errors.New("multiple " + types.ClusterPlaceholder)
	errResSelUnsupported    = errors.New("resource selectors not supported")
)

func wrapKustSrcErr(err error) error {
	return fmt.Errorf("kustomization source error: %w", err)
}

var errAbsPath = errors.New("absolute path not allowed")

// REVISIT: fix cyclop
//
//nolint:cyclop
func (kust *Kustomize) packages(env *types.Env) (*kustomizePkg, error) {
	var pathPrefix, pathSuffix string

	path := kust.PathTemplate
	if filepath.IsAbs(path) {
		return nil, fmt.Errorf("invalid kustomization path: %w", errAbsPath)
	}

	pathParts := strings.Split(path, types.ClusterPlaceholder)

	switch {
	case len(pathParts) == 1 && len(kust.Clusters) == 0:
		idx := types.NewClusterIndex()
		clusterID := idx.Add(types.Cluster{})

		kpkg := &kustomizePkg{
			idx: idx,
			paths: map[types.ClusterID]string{
				clusterID: path,
			},
		}

		return kpkg, nil
	case len(pathParts) < 2 && len(kust.Clusters) > 0:
		return nil, wrapKustSrcErr(errPlaceholderMissing)
	case len(pathParts) > 2: //nolint:mnd
		return nil, wrapKustSrcErr(errMultiplePlaceholders)
	case len(pathParts) == 2: //nolint:mnd
		pathPrefix = pathParts[0]
		pathSuffix = pathParts[1]
	default:
		pathPrefix = pathParts[0]
	}

	pathPattern := strings.Join(pathParts, "*")

	paths, err := env.FileSys.Glob(pathPattern)
	if err != nil {
		return nil, wrapKustSrcErr(err)
	}

	names := []string{}

	for _, foundPath := range paths {
		name := strings.TrimPrefix(foundPath, pathPrefix)
		name = strings.TrimSuffix(name, pathSuffix)
		names = append(names, name)
	}

	kpkg := &kustomizePkg{
		idx:   types.BuildClusterIndex(names, kust.Clusters),
		paths: map[types.ClusterID]string{},
	}

	for clusterID, cluster := range kpkg.idx.All() {
		envPath := pathPrefix + cluster.Name + pathSuffix

		absPath, name, err := env.FileSys.CleanedAbs(envPath)
		if err != nil {
			return nil, err
		}

		kpkg.paths[clusterID] = filepath.Join(string(absPath), name)
	}

	return kpkg, nil
}

func (kust *Kustomize) Load(env *types.Env) (*State, error) {
	if len(kust.Resources) > 0 {
		//nolint:godox
		// TODO: convert to filters
		//	requires api-resource/kind mapping
		//	can be obtained by parsing table from kubectl api-resources
		return nil, wrapKustSrcErr(errResSelUnsupported)
	}

	pkgs, err := kust.packages(env)
	if err != nil {
		return nil, err
	}

	errg := errgroup.Group{}
	buffers := map[types.ClusterID]*kio.PackageBuffer{}

	for clusterID, path := range pkgs.paths {
		buffer := &kio.PackageBuffer{}
		buffers[clusterID] = buffer

		errg.Go(func() error {
			rnodes, err := env.Cmd.BuildKustomization(path)
			if err != nil {
				return err //nolint:wrapcheck
			}

			buffer.Nodes = rnodes

			return nil
		})
	}

	if err := errg.Wait(); err != nil {
		return nil, wrapKustSrcErr(err)
	}

	resources := map[types.ClusterID][]*yaml.RNode{}

	for clusterID, buffer := range buffers {
		resources[clusterID] = buffer.Nodes
	}

	state := &State{pkgs.idx, resources}

	return state, nil
}
