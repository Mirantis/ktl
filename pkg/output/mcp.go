package output

import (
	"fmt"
	"strings"

	"github.com/Mirantis/ktl/pkg/apis"
	"github.com/Mirantis/ktl/pkg/types"
)

func newMCPToolOutput(spec *apis.MCPToolOutput) (*MCPToolOutput, error) {
	impl := &MCPToolOutput{
		Description: spec.GetDescription(),
	}

	for _, colSpec := range spec.GetColumns() {
		ref, err := newValueRef(colSpec)
		if err != nil {
			return nil, err
		}
		impl.Columns = append(impl.Columns, ref)
	}

	return impl, nil

}

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
