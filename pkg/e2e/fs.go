package e2e

import (
	"io/fs"

	"sigs.k8s.io/kustomize/kyaml/filesys"
)

func ReadFiles(f filesys.FileSystem, base string) map[string]string {
	got := map[string]string{}
	f.Walk(base, func(path string, info fs.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		data, err := f.ReadFile(path)
		if err != nil {
			return err
		}
		got[path[len(base):]] = string(data)
		return nil
	})
	return got
}
