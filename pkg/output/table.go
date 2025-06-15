package output

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"path/filepath"

	"github.com/Mirantis/ktl/pkg/apis"
	"github.com/Mirantis/ktl/pkg/types"
	"k8s.io/cli-runtime/pkg/printers"
)

func newTableOutput(spec *apis.ColumnarFileOutput) (*TableOutput, error) {
	impl, err := newCSVOutput(spec)
	if err != nil {
		return nil, err
	}

	return &TableOutput{*impl}, nil
}

type TableOutput struct {
	CSVOutput `yaml:",inline"`
}

func (out *TableOutput) Store(env *types.Env, resources *types.ClusterResources) error {
	path := out.Path
	if filepath.IsAbs(path) {
		return fmt.Errorf("invalid table output path: %w", errAbsPath)
	}

	buffer := bytes.NewBuffer(nil)

	err := func() error {
		tabWriter := printers.GetNewTabWriter(buffer)
		defer tabWriter.Flush()

		csvWriter := csv.NewWriter(tabWriter)
		csvWriter.Comma = '\t'
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
