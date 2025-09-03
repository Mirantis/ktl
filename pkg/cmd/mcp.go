package cmd

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/Mirantis/ktl/pkg/apis"
	"github.com/Mirantis/ktl/pkg/fsutil"
	"github.com/Mirantis/ktl/pkg/kubectl"
	"github.com/Mirantis/ktl/pkg/runner"
	"github.com/Mirantis/ktl/pkg/types"
	"github.com/modelcontextprotocol/go-sdk/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func newMCPCommand() *cobra.Command {
	//TODO: add e2e tests

	toolPaths := []string{}
	resourcePaths := []string{}

	mcpCmd := &cobra.Command{
		Use:   "mcp",
		Short: "run the MCP server",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			srv := mcp.NewServer(
				&mcp.Implementation{
					Name:    "ktl",
					Version: "v1.0-beta1",
				}, nil)

			for _, toolPath := range toolPaths {
				toolSpec, err := loadPipelineSpec(toolPath)
				if err != nil {
					return fmt.Errorf("unable to load %s: %w", toolPath, err)
				}

				if toolSpec.GetName() == "" {
					return fmt.Errorf("missing name for tool %s", toolPath)
				}

				tool := &mcp.Tool{
					Name:        toolSpec.Name,
					Description: toolSpec.GetDescription(),
				}

				if argsSchema := toolSpec.GetArgs().GetSchema(); argsSchema != nil {
					schemaBody, err := json.Marshal(argsSchema)
					if err != nil {
						return err
					}

					tool.InputSchema = &jsonschema.Schema{}
					if err := json.Unmarshal(schemaBody, tool.InputSchema); err != nil {
						return fmt.Errorf("invalid args schema: %w", err)
					}
				}

				if outSchema := toolSpec.GetOutput().GetJson().GetSchema(); outSchema != nil {
					schemaBody, err := json.Marshal(outSchema)
					if err != nil {
						return err
					}

					tool.OutputSchema = &jsonschema.Schema{}
					if err := json.Unmarshal(schemaBody, tool.OutputSchema); err != nil {
						return fmt.Errorf("invalid output schema: %w", err)
					}
				}

				mcp.AddTool(srv, tool, newMCPHandler[map[string]any](
					filepath.Dir(toolPath),
					toolSpec,
				))
			}

			transport := mcp.NewStdioTransport()
			ctx := cmp.Or(cmd.Context(), context.Background())

			return srv.Run(ctx, transport)
		},
	}

	mcpCmd.Flags().StringSliceVarP(
		&toolPaths, "tool", "t", nil,
		"pipeline(s) to be published as an MCP tool",
	)
	mcpCmd.Flags().StringSliceVarP(
		&resourcePaths, "resource", "r", nil,
		"pipeline(s) to be published as an MCP resource",
	)

	return mcpCmd
}

func newMCPHandler[In any](workdir string, spec *apis.Pipeline) mcp.ToolHandlerFor[In, map[string]any] {
	return func(ctx context.Context, ss *mcp.ServerSession, ctpf *mcp.CallToolParamsFor[In]) (*mcp.CallToolResultFor[map[string]any], error) {
		result := &mcp.CallToolResultFor[map[string]any]{
			Content:           []mcp.Content{},
			StructuredContent: map[string]any{},
		}
		stdout := bytes.NewBuffer(nil)
		fileSys := fsutil.Stdio(
			fsutil.Sub(filesys.MakeFsOnDisk(), workdir),
			bytes.NewBuffer(nil),
			stdout,
		)
		env := &types.Env{
			WorkDir: workdir,
			FileSys: fileSys,
			Cmd:     kubectl.New(),
		}

		args := yaml.NewMapRNode(nil)
		if ctpf != nil {
			if err := args.YNode().Encode(ctpf.Arguments); err != nil {
				return nil, err
			}
		}

		pipeline, err := runner.NewPipeline(spec, args)
		if err != nil {
			return nil, err
		}

		if err := pipeline.Run(env); err != nil {
			return nil, err
		}

		if stdout.Len() > 0 {
			//TODO: move extension handling to method
			switch {
			case spec.GetOutput().GetCsv() != nil:
				result.Content = append(result.Content, &mcp.TextContent{
					Text: "<source>result.csv</source>" + stdout.String(),
				})
			case spec.GetOutput().GetCrdDescriptions() != nil:
				result.Content = append(result.Content, &mcp.TextContent{
					Text: "<source>result.json</source>" + stdout.String(),
				})
			case spec.GetOutput().GetJson() != nil:
				err := json.Unmarshal(stdout.Bytes(), &result.StructuredContent)
				if err != nil {
					return nil, err
				}
			default:
				result.Content = append(result.Content, &mcp.TextContent{
					Text: stdout.String(),
				})
			}
		}

		return result, nil
	}
}
