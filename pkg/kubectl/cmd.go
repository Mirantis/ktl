package kubectl

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"k8s.io/kubectl/pkg/cmd/version"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func DefaultCmd() KubectlCmd {
	return []string{"kubectl"}
}

type KubectlCmd []string

func (kc KubectlCmd) String() string {
	return strings.Join(kc, " ")
}

func (kc KubectlCmd) Server(server string) KubectlCmd {
	return append(kc, "--server", server)
}

func (kc KubectlCmd) output(args ...string) ([]byte, error) {
	args = append(kc[1:], args...)
	cmd := exec.Command(kc[0], args...)
	data, err := cmd.Output()

	switch err := err.(type) {
	case nil:
		return data, nil
	case *exec.ExitError:
		stderr := strings.TrimSpace(string(err.Stderr))
		return nil, fmt.Errorf("%s failed: %v, stderr: %s", kc, err, stderr)
	default:
		return nil, err
	}
}

func (kc KubectlCmd) Version(server bool) (*version.Version, error) {
	args := []string{"version", "-ojson"}
	if !server {
		args = append(args, "--client=true")
	}

	data, err := kc.output(args...)
	if err != nil {
		return nil, err
	}

	v := &version.Version{}
	if err := json.Unmarshal(data, v); err != nil {
		return nil, err
	}

	return v, nil
}

func (kc KubectlCmd) ApplyKustomization(path string) error {
	_, err := kc.output("apply", "--kustomize", path)
	return err
}

func (kc KubectlCmd) Get(resources string) ([]*yaml.RNode, error) {
	response, err := kc.output("get", "-oyaml", resources)
	if err != nil {
		return nil, err
	}
	root, err := yaml.Parse(string(response))
	if err != nil {
		return nil, err
	}
	items, err := root.Pipe(yaml.Lookup("items"))
	if err != nil {
		return nil, err
	}
	nodes := []*yaml.RNode{}
	for _, item := range items.Content() {
		nodes = append(nodes, yaml.NewRNode(item))
	}
	return nodes, nil
}
