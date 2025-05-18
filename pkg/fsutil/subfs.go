package fsutil

import (
	"io/fs"
	"path/filepath"

	"sigs.k8s.io/kustomize/kyaml/filesys"
)

type subFileSystem struct {
	delegate filesys.FileSystem
	root     string
}

func Sub(fsys filesys.FileSystem, dir string) filesys.FileSystem {
	for {
		subfs, ok := fsys.(*subFileSystem)
		if !ok {
			break
		}
		fsys = subfs.delegate
		dir = filepath.Join(subfs.root, dir)
	}

	return &subFileSystem{delegate: fsys, root: dir}
}

func (fsys *subFileSystem) path(path string) string {
	delegatePath := path
	if !filepath.IsAbs(path) {
		delegatePath = filepath.Clean(filepath.Join(fsys.root, path))
	}

	return delegatePath
}

func (fsys *subFileSystem) Create(path string) (filesys.File, error) {
	return fsys.delegate.Create(fsys.path(path))
}

func (fsys *subFileSystem) Mkdir(path string) error {
	return fsys.delegate.Mkdir(fsys.path(path))
}

func (fsys *subFileSystem) MkdirAll(path string) error {
	return fsys.delegate.MkdirAll(fsys.path(path))
}

func (fsys *subFileSystem) RemoveAll(path string) error {
	return fsys.delegate.RemoveAll(fsys.path(path))
}

func (fsys *subFileSystem) Open(path string) (filesys.File, error) {
	return fsys.delegate.Open(fsys.path(path))
}

func (fsys *subFileSystem) IsDir(path string) bool {
	return fsys.delegate.IsDir(fsys.path(path))
}

func (fsys *subFileSystem) ReadDir(path string) ([]string, error) {
	return fsys.delegate.ReadDir(fsys.path(path))
}

func (fsys *subFileSystem) CleanedAbs(path string) (filesys.ConfirmedDir, string, error) {
	return fsys.delegate.CleanedAbs(fsys.path(path))
}

func (fsys *subFileSystem) Exists(path string) bool {
	return fsys.delegate.Exists(fsys.path(path))
}

func (fsys *subFileSystem) Glob(pattern string) ([]string, error) {
	matches, err := fsys.delegate.Glob(fsys.path(pattern))
	if err != nil {
		return nil, err
	}

	for i := range matches {
		matches[i], err = filepath.Rel(fsys.root, matches[i])
		if err != nil {
			return nil, err
		}
	}

	return matches, nil
}

func (fsys *subFileSystem) ReadFile(path string) ([]byte, error) {
	return fsys.delegate.ReadFile(fsys.path(path))
}

func (fsys *subFileSystem) WriteFile(path string, data []byte) error {
	return fsys.delegate.WriteFile(fsys.path(path), data)
}

func (fsys *subFileSystem) Walk(path string, walkFn filepath.WalkFunc) error {
	if filepath.IsAbs(path) {
		return fsys.delegate.Walk(path, walkFn)
	}

	realPath := fsys.path(path)

	return fsys.delegate.Walk(realPath, func(path string, info fs.FileInfo, err error) error {
		walkPath, err := filepath.Rel(realPath, path)
		if err != nil {
			return err
		}

		return walkFn(walkPath, info, err)
	})
}
