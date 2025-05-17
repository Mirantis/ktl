package types

import (
	"github.com/Mirantis/ktl/pkg/kubectl"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

type Env struct {
	WorkDir string
	Cmd     *kubectl.Cmd
	FileSys filesys.FileSystem
}
