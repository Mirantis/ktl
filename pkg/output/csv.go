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

func (ref *ValueRef) text(cluster *types.Cluster) string {
	if len(ref.Text) > 0 {
		return strings.ReplaceAll(ref.Text, types.ClusterPlaceholder, cluster.Name)
	}

	return ""
}

type CSVOutput struct {
	Columns []ValueRef `yaml:"columns"`
	Path    string     `yaml:"path"`
}

func (out *CSVOutput) initRow(offset int, cluster *types.Cluster) ([]string, *resource.Queries[int], []int) {
	row := make([]string, len(out.Columns))
	offsets := make([]int, len(out.Columns))
	queries := &resource.Queries[int]{}

	for colIdx, col := range out.Columns {
		row[colIdx] = col.text(cluster)
		offsets[colIdx] = offset

		if len(col.Field) == 0 {
			continue
		}

		queries.Add(col.Field, colIdx)
	}

	return row, queries, offsets
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
			cluster := resources.Clusters.Cluster(clusterID)
			row, queries, offsets := out.initRow(len(rows)-1, &cluster)

			for colIdx, valueNode := range queries.Scan(node) {
				value, _ := valueNode.String()
				value = strings.TrimSpace(value)

				if offsets[colIdx] > len(rows)-1 {
					rows = append(rows, slices.Clone(row))
					for oIdx := range offsets {
						offsets[oIdx] = len(rows) - 1
					}
				}

				row[colIdx] = value
				offsets[colIdx]++
			}

			for colIdx := range offsets {
				if offsets[colIdx] > len(rows)-1 {
					rows = append(rows, slices.Clone(row))
					break
				}
			}
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
