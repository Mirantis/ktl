package resource

import (
	"bytes"
	"fmt"
	"iter"
	"path/filepath"
	"strings"

	"github.com/Mirantis/rekustomize/pkg/types"
	"sigs.k8s.io/kustomize/api/konfig"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/resid"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type FileStore struct {
	filesys.FileSystem
	NameGenerator func(resid.ResId) string
	PostProcessor func(string, []byte) []byte
}

func (store *FileStore) WriteAll(nodes iter.Seq2[resid.ResId, *yaml.RNode]) error {
	buf := &bytes.Buffer{}

	for resID, resNode := range nodes {
		path := store.NameGenerator(resID)
		if err := store.MkdirAll(filepath.Dir(path)); err != nil {
			return fmt.Errorf("unable to initialize dir for %v: %w", path, err)
		}

		buf.Reset()

		err := kio.ByteWriter{
			Writer: buf,

			ClearAnnotations: []string{kioutil.PathAnnotation},
		}.Write([]*yaml.RNode{resNode})
		if err != nil {
			return fmt.Errorf("unable to serialize %v: %w", path, err)
		}

		body := buf.Bytes()
		if store.PostProcessor != nil {
			body = store.PostProcessor(path, body)
		}

		if err := store.WriteFile(path, body); err != nil {
			return fmt.Errorf("unable to write %v: %w", path, err)
		}
	}

	return nil
}

func (store *FileStore) WriteKustomization(kust *types.Kustomization) error {
	kustBytes, err := yaml.Marshal(kust)
	if err != nil {
		return fmt.Errorf("unable to generate kustomization.yaml: %w", err)
	}

	if err := store.MkdirAll("."); err != nil {
		return fmt.Errorf("unable to initialize dir: %w", err)
	}

	kustPath := konfig.DefaultKustomizationFileName()
	if err := store.WriteFile(kustPath, kustBytes); err != nil {
		return fmt.Errorf("unable to store %q: %w", kustPath, err)
	}

	return nil
}

func FileName(resID resid.ResId) string {
	parts := []string{}
	if resID.Namespace != "" {
		parts = append(parts, resID.Namespace)
	}

	parts = append(parts, strings.ToLower(fmt.Sprintf("%s-%s.yaml", resID.Name, resID.Kind)))

	return filepath.Join(parts...)
}
