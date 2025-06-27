package output

import (
	"bytes"
	_ "embed"
	"io"
	"testing"

	"github.com/Mirantis/ktl/pkg/fsutil"
	"github.com/Mirantis/ktl/pkg/types"
	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/resid"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var (
	//go:embed testdata/crds/clustertemplates.yaml
	clusterTemplatesCrd []byte
	//go:embed testdata/crds/servicetemplates.yaml
	serviceTemplateCrd []byte

	//go:embed testdata/crds/summary.json
	crdSummaryJson []byte
)

func TestCrdSummaryOutput(t *testing.T) {
	var rnode *yaml.RNode
	resid.FromRNode(rnode)
	r := kio.ByteReader{
		Reader: io.MultiReader(
			bytes.NewBuffer(clusterTemplatesCrd),
			bytes.NewBuffer([]byte{'\n'}),
			bytes.NewBuffer(serviceTemplateCrd),
		),
	}

	rnodes, err := r.Read()
	if err != nil {
		t.Fatal(err)
	}

	cres := &types.ClusterResources{
		Resources: map[resid.ResId]map[types.ClusterID]*yaml.RNode{},
	}

	for _, rnode := range rnodes {
		rid := resid.FromRNode(rnode)
		cres.Resources[rid] = map[types.ClusterID]*yaml.RNode{0: rnode}
	}

	stdout := bytes.NewBuffer(nil)
	out := &CRDSummaryOutput{}
	fileSys := fsutil.Stdio(
		filesys.MakeFsInMemory(),
		bytes.NewBuffer(nil),
		stdout,
	)
	env := &types.Env{
		FileSys: fileSys,
	}

	if err := out.Store(env, cres); err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(crdSummaryJson[:len(crdSummaryJson)-1], stdout.Bytes()); diff != "" {
		t.Errorf("-want +got:\n%s", diff)
	}
}
