package helm

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestYamlCommentsStillBroken(t *testing.T) {
	const (
		input = `root1:
- node: a
  value: 1
  opt: true
- node: b
  value: 2
root2: {}
`
		want = `root1:
# head0
- node: a
  value: 1
  # head1
  opt: true # line1
  # foot1
# head2
-
# foot0
  node: b
  value: 2
root2:
# foot2
{}
`
	)

	rn := yaml.MustParse(input)
	yn := rn.YNode()
	yn.Content[1].Content[0].HeadComment = "head0"
	yn.Content[1].Content[0].FootComment = "foot0"
	yn.Content[1].Content[0].Content[4].HeadComment = "head1"
	yn.Content[1].Content[0].Content[4].FootComment = "foot1"
	yn.Content[1].Content[0].Content[4].LineComment = "line1"
	yn.Content[1].Content[1].HeadComment = "head2"
	yn.Content[1].Content[1].FootComment = "foot2"
	got, err := rn.String()
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("-want +got:\n%v", diff)
	}
}

func TestYamlCommentsWorkaround(t *testing.T) {
	const (
		input = `root1:
- node: a
  value: 1
  opt: true
- node: b
  value: 2
root2: {}
`
		want = `root1:
# head0
- node: a
  value: 1
  # head1
  opt: true # line1
  # foot1
# foot0
# head2
- node: b
  value: 2
# foot2

root2: {}
`
	)

	rn := yaml.MustParse(input)
	if err := rn.PipeE(yaml.Lookup("root1", "[node=a]"), setComments("head0", "foot0")); err != nil {
		t.Fatal(err)
	}
	if err := rn.PipeE(yaml.Lookup("root1", "[node=a]", "opt"), setComments("head1", "line1", "foot1")); err != nil {
		t.Fatal(err)
	}
	if err := rn.PipeE(yaml.Lookup("root1", "[node=b]"), setComments("head2", "foot2")); err != nil {
		t.Fatal(err)
	}
	fixComments(rn.YNode())
	got, err := rn.String()
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("-want +got:\n%v", diff)
	}
}
