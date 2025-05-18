package fsutil

import (
	"bytes"
	"io"
	"path/filepath"

	"sigs.k8s.io/kustomize/kyaml/filesys"
)

type stdioFileSystem struct {
	delegate filesys.FileSystem
	stdin    io.Reader
	stdout   io.Writer
}

func Stdio(fsys filesys.FileSystem, stdin io.Reader, stdout io.Writer) filesys.FileSystem {
	return &stdioFileSystem{
		delegate: fsys,
		stdin:    stdin,
		stdout:   stdout,
	}
}

func (fsys *stdioFileSystem) Create(path string) (filesys.File, error) {
	return fsys.delegate.Create(path)
}

func (fsys *stdioFileSystem) Mkdir(path string) error {
	return fsys.delegate.Mkdir(path)
}

func (fsys *stdioFileSystem) MkdirAll(path string) error {
	return fsys.delegate.MkdirAll(path)
}

func (fsys *stdioFileSystem) RemoveAll(path string) error {
	return fsys.delegate.RemoveAll(path)
}

func (fsys *stdioFileSystem) Open(path string) (filesys.File, error) {
	return fsys.delegate.Open(path)
}

func (fsys *stdioFileSystem) IsDir(path string) bool {
	return fsys.delegate.IsDir(path)
}

func (fsys *stdioFileSystem) ReadDir(path string) ([]string, error) {
	return fsys.delegate.ReadDir(path)
}

func (fsys *stdioFileSystem) CleanedAbs(path string) (filesys.ConfirmedDir, string, error) {
	return fsys.delegate.CleanedAbs(path)
}

func (fsys *stdioFileSystem) Exists(path string) bool {
	return fsys.delegate.Exists(path)
}

func (fsys *stdioFileSystem) Glob(pattern string) ([]string, error) {
	return fsys.delegate.Glob(pattern)
}

func (fsys *stdioFileSystem) ReadFile(path string) ([]byte, error) {
	if path == "-" || len(path) == 0 {
		return io.ReadAll(fsys.stdin)
	}

	return fsys.delegate.ReadFile(path)
}

func (fsys *stdioFileSystem) WriteFile(path string, data []byte) error {
	if path == "-" || len(path) == 0 {
		_, err := io.Copy(fsys.stdout, bytes.NewReader(data))
		return err
	}

	return fsys.delegate.WriteFile(path, data)
}

func (fsys *stdioFileSystem) Walk(path string, walkFn filepath.WalkFunc) error {
	return fsys.delegate.Walk(path, walkFn)
}
