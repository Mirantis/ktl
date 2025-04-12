package config

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/Mirantis/rekustomize/pkg/helm"
	"github.com/Mirantis/rekustomize/pkg/kustomize"
	"github.com/Mirantis/rekustomize/pkg/resource"
	"github.com/Mirantis/rekustomize/pkg/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

const (
	dirPerm = 0o700
)

type Output struct {
	WorkDir string
	FileSys filesys.FileSystem

	HelmChart types.HelmChart `yaml:"helmChart"`

	resources *types.ClusterResources
	chartDir  string
	compsDir  string
}

func (out *Output) Store(resources *types.ClusterResources) error {
	out.resources = resources
	if len(resources.Clusters.IDs()) == 1 {
		return out.runSingle()
	}

	if len(out.HelmChart.Name) == 0 {
		return out.exportComponents()
	}

	return out.exportCharts()
}

func (out *Output) storeChartOverlays(chart *helm.Chart) error {
	for clusterID, cluster := range out.resources.Clusters.All() {
		fileStore := resource.FileStore{
			Dir:        filepath.Join(out.WorkDir, "overlays", cluster.Name),
			FileSystem: filesys.FileSystemOrOnDisk{FileSystem: filesys.MakeFsOnDisk()},
		}
		kust := &types.Kustomization{}
		kust.Kind = types.KustomizationKind

		chartHome, err := filepath.Rel(fileStore.Dir, filepath.Dir(out.chartDir))
		if err != nil {
			return fmt.Errorf("invalid path: %w", err)
		}

		kust.HelmCharts = []types.HelmChart{chart.Instance(clusterID)}
		kust.HelmGlobals = &types.HelmGlobals{
			ChartHome: chartHome,
		}

		if err := fileStore.WriteKustomization(kust); err != nil {
			return fmt.Errorf("unable to store kustomization: %w", err)
		}
	}

	return nil
}

func (out *Output) exportCharts() error {
	chartMeta := out.HelmChart
	chart := helm.NewChart(chartMeta, out.resources.Clusters)
	out.chartDir = filepath.Join(out.WorkDir, "charts", chartMeta.Name)

	if err := os.MkdirAll(out.chartDir, dirPerm); err != nil {
		return fmt.Errorf("unable to create %v: %w", out.chartDir, err)
	}

	for id, byCluster := range out.resources.Resources {
		if err := chart.Add(id, byCluster); err != nil {
			return fmt.Errorf("unable to add resources to the chart: %w", err)
		}
	}

	diskFs := filesys.MakeFsOnDisk()
	if err := chart.Store(diskFs, out.chartDir); err != nil {
		return fmt.Errorf("unable to store the chart: %w", err)
	}

	return out.storeChartOverlays(chart)
}

func (out *Output) storeComponentsOverlays(comps *kustomize.Components) error {
	for clusterID, cluster := range out.resources.Clusters.All() {
		fileStore := resource.FileStore{
			Dir:        filepath.Join(out.WorkDir, "overlays", cluster.Name),
			FileSystem: filesys.FileSystemOrOnDisk{FileSystem: filesys.MakeFsOnDisk()},
		}
		kust := &types.Kustomization{}
		kust.Kind = types.KustomizationKind

		compNames, err := comps.Cluster(clusterID)
		if err != nil {
			panic(err)
		}

		for _, compName := range compNames {
			relPath, err := filepath.Rel(fileStore.Dir, filepath.Join(out.compsDir, compName))
			if err != nil {
				panic(err)
			}

			kust.Components = append(kust.Components, relPath)
		}

		if err := fileStore.WriteKustomization(kust); err != nil {
			return fmt.Errorf("unable to store kustomization: %w", err)
		}
	}

	return nil
}

func (out *Output) exportComponents() error {
	comps := kustomize.NewComponents(out.resources.Clusters)
	out.compsDir = filepath.Join(out.WorkDir, "components")

	for id, byCluster := range out.resources.Resources {
		if err := comps.Add(id, byCluster); err != nil {
			return fmt.Errorf("unable to add resources to the component: %w", err)
		}
	}

	diskFs := filesys.MakeFsOnDisk()
	if err := comps.Store(diskFs, out.compsDir); err != nil {
		return fmt.Errorf("unable to store components: %w", err)
	}

	return out.storeComponentsOverlays(comps)
}

func (out *Output) runSingle() error {
	kust := &types.Kustomization{}
	resourceStore := &resource.FileStore{
		Dir:           out.WorkDir,
		FileSystem:    filesys.FileSystemOrOnDisk{FileSystem: filesys.MakeFsOnDisk()},
		NameGenerator: resource.FileName,
		PostProcessor: func(path string, body []byte) []byte {
			relPath, err := filepath.Rel(out.WorkDir, path)
			if err != nil {
				panic(err)
			}
			kust.Resources = append(kust.Resources, relPath)

			return body
		},
	}

	if err := resourceStore.WriteAll(out.resources.All()); err != nil {
		return fmt.Errorf("unable to store files: %w", err)
	}

	slices.Sort(kust.Resources)

	if err := resourceStore.WriteKustomization(kust); err != nil {
		return fmt.Errorf("unable to store kustomization: %w", err)
	}

	return nil
}
