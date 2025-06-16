package cmd

import (
	"bytes"
	"os"
	"strings"
	"text/template"

	"github.com/Mirantis/ktl/pkg/runner"
	"github.com/spf13/cobra"

	"github.com/Mirantis/ktl/pkg/fsutil"
	"github.com/Mirantis/ktl/pkg/kubectl"
	"github.com/Mirantis/ktl/pkg/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var (
	queryPipelineTmpl = template.Must(template.New("query").Parse(queryPipelineBody))
	queryPipelineBody = `
source:
  clusters:
  - names: {{ .clusters | printf "%q" }}
  resources:
  - apiResources: {{ .resources | printf "%q" }}
filters:
- kind: Starlark
  script: |-
    for it in resources:
      if {{ .query }}:
        output.append(it)
output:
  kind: Table
  columns:
  - name: CLUSTER
    text: "${CLUSTER}"
  - name: KIND
    field: kind
  - name: NAMESPACE
    field: metadata.namespace
  - name: NAME
    field: metadata.name
`
)

func newQueryCommand() *cobra.Command {
	clusters := []string{}

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

			pipelineBuf := bytes.NewBuffer(nil)

			err := queryPipelineTmpl.Execute(pipelineBuf, map[string]string{
				"clusters":  strings.Join(clusters, ","),
				"resources": resources,
				"query":     strings.Join(query, " "),
			})
			if err != nil {
				return err
			}

			pipeline := &runner.Pipeline{}
			if err := yaml.Unmarshal(pipelineBuf.Bytes(), pipeline); err != nil {
				return err
			}

			return pipeline.Run(env)
		},
	}

	export.Flags().StringSliceVar(&clusters, "clusters", []string{"*"}, "clusters pattern")

	return export
}
