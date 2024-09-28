package kubectl

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"k8s.io/kubectl/pkg/cmd/version"
)

const DefaultCmd KubectlCmd = "kubectl"

type KubectlCmd string

func (kc KubectlCmd) output(args ...string) ([]byte, error) {
	cmd := exec.Command(string(kc), args...)
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
