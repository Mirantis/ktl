package cmd

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/Mirantis/ktl/pkg/apis"
	"github.com/Mirantis/ktl/pkg/output"
	"github.com/Mirantis/ktl/pkg/runner"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"
	"sigs.k8s.io/yaml"
)

func newMCPCommand() *cobra.Command {
	mcpCmd := &cobra.Command{
		Use:   "mcp",
		Short: "LLM integration (in development)",
	}
	mcpCmd.AddCommand(newMCPDescribeCommand())

	return mcpCmd
}

func newMCPDescribeCommand() *cobra.Command {
	export := &cobra.Command{
		Use:   "describe FILENAME",
		Short: "describe the report",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fileName := args[0]

			pipelineBytes, err := os.ReadFile(fileName)
			if err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("unable to read %s: %w", fileName, err)
			}
			slog.Info("mcp describe", "pipeline", pipelineBytes, "file", fileName)

			pipelineJSON, err := yaml.YAMLToJSON(pipelineBytes)
			if err != nil {
				return err
			}

			pipelineSpec := &apis.Pipeline{}

			if err := protojson.Unmarshal(pipelineJSON, pipelineSpec); err != nil {
				return err
			}

			pipeline, err := runner.NewPipeline(pipelineSpec)
			if err != nil {
				return err
			}

			mcptool, ok := pipeline.Output.Impl.(*output.MCPToolOutput)
			if !ok {
				return fmt.Errorf("output kind must be MCPTool")
			}

			_, err = io.WriteString(cmd.OutOrStdout(), mcptool.Describe())

			return err
		},
	}

	return export
}
