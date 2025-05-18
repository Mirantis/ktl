package output

import (
	"fmt"
	"strings"

	"github.com/Mirantis/ktl/pkg/types"
)

type MCPToolOutput struct {
	Columns     []ValueRef `yaml:"columns"`
	Description string     `yaml:"description"`
}

func (out *MCPToolOutput) Store(env *types.Env, resources *types.ClusterResources) error {
	csvOut := &CSVOutput{Columns: out.Columns}
	return csvOut.Store(env, resources)
}

func (out *MCPToolOutput) Describe() string {
	parts := []string{out.Description, "", "Columns:"}

	for _, column := range out.Columns {
		parts = append(parts, fmt.Sprintf(
			"- %s: %s",
			column.Name,
			column.Description,
		))
	}

	parts = append(parts, "")

	return strings.Join(parts, "\n")
}
