package cmd

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
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
			srv := mcp.NewServer("ktl", "v1.0-beta1", nil)
			tools := []*mcp.ServerTool{}

			for _, toolPath := range toolPaths {
				toolSpec, err := loadPipelineSpec(toolPath)
				if err != nil {
					return fmt.Errorf("unable to load %s: %w", toolPath, err)
				}

				if toolSpec.GetName() == "" {
					return fmt.Errorf("missing name for tool %s", toolPath)
				}

				if inlineSchema := toolSpec.GetArgs().GetSchema(); inlineSchema != nil {
					tool := mcp.NewServerTool(
						toolSpec.Name,
						toolSpec.GetDescription(),
						newMCPHandler[map[string]any](filepath.Dir(toolPath), toolSpec),
					)
					schemaBody, err := json.Marshal(inlineSchema)
					if err != nil {
						return err
					}
					argsSchema := &jsonschema.Schema{}
					if err := json.Unmarshal(schemaBody, argsSchema); err != nil {
						return err
					}
					tool.Tool.InputSchema = argsSchema
					tools = append(tools, tool)
				} else {
					tool := mcp.NewServerTool(
						toolSpec.Name,
						toolSpec.GetDescription(),
						newMCPHandler[struct{}](filepath.Dir(toolPath), toolSpec),
					)
					tools = append(tools, tool)
				}

			}

			srv.AddTools(tools...)

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

func newMCPHandler[In any](workdir string, spec *apis.Pipeline) mcp.ToolHandlerFor[In, struct{}] {
	return func(ctx context.Context, ss *mcp.ServerSession, ctpf *mcp.CallToolParamsFor[In]) (*mcp.CallToolResultFor[struct{}], error) {
		result := &mcp.CallToolResultFor[struct{}]{}
		stdout := bytes.NewBuffer(nil)
		fileSys := fsutil.Stdio(
			filesys.MakeFsInMemory(),
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

		err = env.FileSys.Walk(".", func(path string, info fs.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}
			content, err := env.FileSys.ReadFile(path)
			if err != nil {
				return err
			}

			result.Content = append(result.Content, &mcp.TextContent{
				Text: "<source>" + path + "</source>" + string(content),
			})
			return nil
		})
		if err != nil {
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
			default:
				result.Content = append(result.Content, &mcp.TextContent{
					Text: stdout.String(),
				})
			}
		}

		return result, nil
	}
}
