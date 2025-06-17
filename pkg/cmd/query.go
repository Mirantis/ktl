package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/Mirantis/ktl/pkg/filters"
	"github.com/Mirantis/ktl/pkg/output"
	"github.com/Mirantis/ktl/pkg/resource"
	"github.com/Mirantis/ktl/pkg/runner"
	"github.com/Mirantis/ktl/pkg/source"
	"github.com/spf13/cobra"

	"github.com/Mirantis/ktl/pkg/fsutil"
	"github.com/Mirantis/ktl/pkg/kubectl"
	"github.com/Mirantis/ktl/pkg/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	kfilters "sigs.k8s.io/kustomize/kyaml/kio/filters"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func newQueryCommand() *cobra.Command {
	namespaces := ""
	clusters := ""
	columns := []string{}
	extraColumns := []string{}
	format := ""

	export := &cobra.Command{
		Use:   "query RESOURCES [FILTER]",
		Short: "query cluster resources",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resources, query := args[0], args[1:]
			if len(query) == 0 {
				query = []string{"True"}
			}

			workDir, _ := os.Getwd()
			fileSys := fsutil.Stdio(
				fsutil.Sub(filesys.MakeFsOnDisk(), workDir),
				cmd.InOrStdin(), cmd.OutOrStdout(),
			)
			env := &types.Env{
				WorkDir: workDir,
				FileSys: fileSys,
				Cmd:     kubectl.New(),
			}

			clustersPattern := types.PatternSelector{}
			clustersYNode := yaml.NewStringRNode(clusters).YNode()

			if err := clustersPattern.UnmarshalYAML(clustersYNode); err != nil {
				return err
			}

			resourcesPattern := types.PatternSelector{}
			resourcesYNode := yaml.NewStringRNode(resources).YNode()

			if err := resourcesPattern.UnmarshalYAML(resourcesYNode); err != nil {
				return err
			}

			nsPattern := types.PatternSelector{}
			nsYNode := yaml.NewStringRNode(namespaces).YNode()

			if err := nsPattern.UnmarshalYAML(nsYNode); err != nil {
				return err
			}

			csvOut := output.CSVOutput{}

			pipeline := &runner.Pipeline{
				Source: runner.Source{
					Impl: &source.Kubeconfig{
						Clusters: []types.ClusterSelector{
							{Names: clustersPattern},
						},
						Resources: source.DefaultResources([]types.ResourceSelector{
							{
								Resources:  resourcesPattern,
								Namespaces: nsPattern,
							},
						}),
					},
				},
			}

			if len(columns) == 0 {
				csvOut.Columns = []output.ValueRef{
					{Name: "CLUSTER", Text: "${CLUSTER}"},
					{Name: "KIND", Field: resource.Query{"kind"}},
					{Name: "NAMESPACE", Field: resource.Query{"metadata", "namespace"}},
					{Name: "NAME", Field: resource.Query{"metadata", "name"}},
				}
			}

			for _, pairs := range extraColumns {
				parts := strings.SplitN(pairs, ":", 2)
				if len(parts) < 2 {
					return fmt.Errorf("%s is not a valid column definition", pairs)
				}

				qYNode := yaml.NewStringRNode(parts[1]).YNode()
				q := resource.Query{}

				if err := q.UnmarshalYAML(qYNode); err != nil {
					return err
				}

				csvOut.Columns = append(csvOut.Columns, output.ValueRef{
					Name:  parts[0],
					Field: q,
				})
			}

			if format == "table" {
				pipeline.Output = runner.Output{
					Impl: &output.TableOutput{CSVOutput: csvOut},
				}
			} else {
				pipeline.Output = runner.Output{
					Impl: &csvOut,
				}
			}

			if len(query) > 0 {
				slf := &filters.StarlarkFilter{
					Script: fmt.Sprintf(("" +
						"for it in resources:\n" +
						"  if %v:\n" +
						"    output.append(it)\n"),
						strings.Join(query, " "),
					),
				}

				pipeline.Filters = []kfilters.KFilter{{Filter: slf}}
			}

			return pipeline.Run(env)
		},
	}

	export.Flags().StringVar(&format, "format", "table", "format, one of: table,csv (default: table)")
	export.Flags().StringVar(&clusters, "clusters", "*", "clusters pattern")
	export.Flags().StringSliceVarP(&columns, "columns", "c", []string{}, "columns, comma-separated <NAME>:<QUERY> pairs (default: CLUSTER, KIND, NAMESPACE, NAME)")
	export.Flags().StringSliceVarP(&extraColumns, "extra-columns", "C", []string{}, "additional columns, comma-separated <NAME>:<QUERY> pairs")
	export.Flags().StringVarP(&namespaces, "namespaces", "n", "*", "namespaces pattern (default: all)")

	return export
}
