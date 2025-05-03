package output

import (
	"bytes"
	"encoding/csv"
	"path/filepath"

	"github.com/Mirantis/rekustomize/pkg/types"
	"k8s.io/cli-runtime/pkg/printers"
)

type TableOutput struct {
	CSVOutput `yaml:",inline"`
}

func (out *TableOutput) Store(env *types.Env, resources *types.ClusterResources) error {
	path := out.Path
	if !filepath.IsAbs(path) {
		path = filepath.Clean(filepath.Join(env.WorkDir, path))
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
