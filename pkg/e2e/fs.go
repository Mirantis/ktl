package e2e

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"strings"
	"testing"

	"sigs.k8s.io/kustomize/kyaml/filesys"
)

func ReadFiles(t *testing.T, fileSys filesys.FileSystem, basePath string) map[string]string {
	t.Helper()

	got := map[string]string{}
	walkFn := func(path string, info fs.FileInfo, _ error) error {
		if info.IsDir() {
			return nil
		}

		data, err := fileSys.ReadFile(path)
		if err != nil {
			return fmt.Errorf("unable to read file: %w", err)
		}

		got[path[len(basePath):]] = string(data)

		return nil
	}

	err := fileSys.Walk(basePath, walkFn)
	if err != nil {
		t.Fatal(err)
	}

	return got
}

func ReadFsFiles(t *testing.T, fileSys fs.FS, base string) map[string]string {
	t.Helper()

	got := map[string]string{}
	walkFn := func(path string, info fs.DirEntry, _ error) error {
		if info != nil && info.IsDir() {
			return nil
		}

		file, err := fileSys.Open(path)
		if err != nil {
			return fmt.Errorf("unable to open file: %w", err)
		}
		defer file.Close()

		buffer := &bytes.Buffer{}
		if _, err := io.Copy(buffer, file); err != nil {
			return fmt.Errorf("unable to copy file: %w", err)
		}

		got[strings.TrimPrefix(path, base+"/")] = buffer.String()

		return nil
	}

	err := fs.WalkDir(fileSys, base, walkFn)
	if err != nil {
		t.Fatal(err)
	}

	return got
}
