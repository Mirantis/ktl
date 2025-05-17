package output

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Mirantis/ktl/pkg/resource"
	"github.com/Mirantis/ktl/pkg/types"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type ValueRef struct {
	Name  string         `yaml:"name"`
	Field resource.Query `yaml:"field"`
	Text  string         `yaml:"text"`
}

func (ref *ValueRef) UnmarshalYAML(node *yaml.Node) error {
	type valueRef ValueRef

	raw := &valueRef{}

	if err := node.Decode(raw); err != nil {
		return err //nolint:wrapcheck
	}

	if len(raw.Field) > 0 && len(raw.Text) > 0 {
		return fmt.Errorf("%w: field,text", errMutuallyExclusive)
	}

	*ref = ValueRef(*raw)

	return nil
}

func (ref *ValueRef) resolve(cluster *types.Cluster, node *yaml.RNode) string {
	if len(ref.Text) > 0 {
		return strings.ReplaceAll(ref.Text, types.ClusterPlaceholder, cluster.Name)
	}

	if len(ref.Field) == 0 {
		return ""
	}

	vnode, _ := node.Pipe(yaml.Lookup(ref.Field...))
	if vnode.IsNilOrEmpty() {
		return ""
	}

	return strings.TrimSpace(vnode.MustString())
}

type CSVOutput struct {
	Columns []ValueRef `yaml:"columns"`
	Path    string     `yaml:"path"`
}

func (out *CSVOutput) rows(resources *types.ClusterResources) [][]string {
	rows := [][]string{}

	header := []string{}
	for _, ref := range out.Columns {
		header = append(header, ref.Name)
	}

	rows = append(rows, header)

	for _, byCluster := range resources.Resources {
		for clusterID, node := range byCluster {
			row := make([]string, len(out.Columns))

			cluster := resources.Clusters.Cluster(clusterID)
			for colIdx, ref := range out.Columns {
				row[colIdx] = ref.resolve(&cluster, node)
			}

			rows = append(rows, row)
		}
	}

	slices.SortFunc(rows[1:], func(rowa, rowb []string) int {
		return slices.CompareFunc(rowa, rowb, strings.Compare)
	})

	return rows
}

var errAbsPath = errors.New("absolute path not supported")

func (out *CSVOutput) Store(env *types.Env, resources *types.ClusterResources) error {
	path := out.Path
	if filepath.IsAbs(path) {
		return fmt.Errorf("invalid csv output path: %w", errAbsPath)
	}

	buffer := bytes.NewBuffer(nil)

	err := func() error {
		csvWriter := csv.NewWriter(buffer)
		defer csvWriter.Flush()

		for _, row := range out.rows(resources) {
			if err := csvWriter.Write(row); err != nil {
				return err //nolint:wrapcheck
			}
		}

		return nil
	}()
	if err != nil {
		return err
	}

	return env.FileSys.WriteFile(path, buffer.Bytes()) //nolint:wrapcheck
}
