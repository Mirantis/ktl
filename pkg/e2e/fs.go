package e2e

import (
	"bytes"
	"io"
	"io/fs"
	"strings"

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
func ReadFsFiles(f fs.FS, base string) map[string]string {
	got := map[string]string{}
	fs.WalkDir(f, base, func(path string, info fs.DirEntry, err error) error {
		if info.IsDir() {
			return nil
		}
		file, err := f.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		buffer := &bytes.Buffer{}
		if _, err := io.Copy(buffer, file); err != nil {
			return err
		}
		got[strings.TrimPrefix(path[len(base):], "/")] = buffer.String()
		return nil
	})
	return got
}
