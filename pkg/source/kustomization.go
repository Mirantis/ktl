package source

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Mirantis/rekustomize/pkg/kubectl"
	"github.com/Mirantis/rekustomize/pkg/types"
	"golang.org/x/sync/errgroup"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/resid"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type Kustomize struct {
	idx   *types.ClusterIndex
	cmd   *kubectl.Cmd
	paths map[types.ClusterID]string
}

const clusterPlaceholder = `${CLUSTER}`

var (
	errPlaceholderMissing   = errors.New("missing " + clusterPlaceholder)
	errMultiplePlaceholders = errors.New("multiple " + clusterPlaceholder)
	errResSelUnsupported    = errors.New("resource selectors not supported")
)

func wrapKustSrcErr(err error) error {
	return fmt.Errorf("kustomization source error: %w", err)
}

//nolint:lll
func NewKustomize(cmd *kubectl.Cmd, fileSys filesys.FileSystem, path string, selectors []types.ClusterSelector) (*Kustomize, error) {
	pathParts := strings.Split(path, clusterPlaceholder)

	if len(pathParts) == 1 && len(selectors) == 0 {
		idx := types.NewClusterIndex()
		clusterID := idx.Add(types.Cluster{})

		kust := &Kustomize{
			idx: idx,
			cmd: cmd,
			paths: map[types.ClusterID]string{
				clusterID: path,
			},
		}

		return kust, nil
	}

	if len(pathParts) < 2 && len(selectors) > 0 {
		return nil, wrapKustSrcErr(errPlaceholderMissing)
	}

	if len(pathParts) > 2 { //nolint:mnd
		return nil, wrapKustSrcErr(errMultiplePlaceholders)
	}

	pathPrefix, pathSuffix := pathParts[0], ""
	if len(pathParts) == 2 { //nolint:mnd
		pathSuffix = pathParts[1]
	}

	pathPattern := strings.Join(pathParts, "*")

	paths, err := fileSys.Glob(pathPattern)
	if err != nil {
		return nil, wrapKustSrcErr(err)
	}

	names := []string{}

	for _, foundPath := range paths {
		name := strings.TrimPrefix(foundPath, pathPrefix)
		name = strings.TrimSuffix(name, pathSuffix)
		names = append(names, name)
	}

	kust := &Kustomize{
		idx:   types.BuildClusterIndex(names, selectors),
		cmd:   cmd,
		paths: map[types.ClusterID]string{},
	}

	for clusterID, cluster := range kust.idx.All() {
		kust.paths[clusterID] = pathPrefix + cluster.Name + pathSuffix
	}

	return kust, nil
}

//nolint:lll
func (kust *Kustomize) Resources(selectors []types.ResourceSelector, filters []kio.Filter) (*types.ClusterResources, error) {
	if len(selectors) > 0 {
		//nolint:godox
		// TODO: convert to filters
		//	requires api-resource/kind mapping
		//	can be obtained by parsing table from kubectl api-resources
		return nil, wrapKustSrcErr(errResSelUnsupported)
	}

	cres := &types.ClusterResources{
		Clusters:  kust.idx,
		Resources: map[resid.ResId]map[types.ClusterID]*yaml.RNode{},
	}

	errg := errgroup.Group{}
	clusterRNodes := map[types.ClusterID]*kio.PackageBuffer{}

	for clusterID, path := range kust.paths {
		buffer := &kio.PackageBuffer{}
		clusterRNodes[clusterID] = buffer

		errg.Go(func() error {
			rnodes, err := kust.cmd.BuildKustomization(path)
			if err != nil {
				return err //nolint:wrapcheck
			}

			return kio.Pipeline{
				Inputs:  []kio.Reader{&kio.PackageBuffer{Nodes: rnodes}},
				Filters: filters,
				Outputs: []kio.Writer{buffer},
			}.Execute()
		})
	}

	if err := errg.Wait(); err != nil {
		return nil, wrapKustSrcErr(err)
	}

	for clusterID, buffer := range clusterRNodes {
		for _, rnode := range buffer.Nodes {
			resID := resid.FromRNode(rnode)

			byResID, foundByID := cres.Resources[resID]
			if !foundByID {
				byResID = map[types.ClusterID]*yaml.RNode{}
				cres.Resources[resID] = byResID
			}

			byResID[clusterID] = rnode
		}
	}

	return cres, nil
}
