package filters_test

import (
	"strings"
	"testing"

	"github.com/Mirantis/rekustomize/pkg/filters"
	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestStarlark(t *testing.T) {
	//TODO: split into unit tests
	input := []*yaml.RNode{
		yaml.MustParse(`
a:
  b: 1
  c: 2
  d:
  - k: dk1
    v: dv1
  - k: dk2
    v: dv2
`),
		yaml.MustParse(`
e: f
`),
		yaml.MustParse(`
g: h
`),
	}
	want := `---
a:
  b: 1
  c: 2
  d:
  - k: dk1
    v:
      "k": "new-dv1"
    old: dv1
  - k: dk2
    v:
      "k": "new-dv2"
    old: dv2
    match: "found"
---
e: f
---
g: h
i: 1
`
	filter := &filters.StarlarkFilter{
		Script: `
for r in resources:
  if r["g"]:
    r.i = 1
  if not r["a.d"]:
    continue
  for d in r.a.d:
	d.old = d.v
	d.v = { "k": "new-%s"%(d["v"]) }
  for m in r["a.d.[old=dv2]"]:
	m.match = "found"
  for nm in r["a.d.[k=dk3]"]:
    nm.nomatch = "found"
`,
	}

	rnodes, err := filter.Filter(input)
	if err != nil {
		t.Fatal(err)
	}

	gotStrings := []string{""}
	for _, rnode := range rnodes {
		gotStrings = append(gotStrings, rnode.MustString())
	}

	got := strings.Join(gotStrings, "---\n")

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("+got -want:\n%s", diff)
	}
}
