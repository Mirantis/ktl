package kubectl

import (
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"slices"
	"strconv"
	"strings"

	"github.com/Mirantis/rekustomize/pkg/types"
	"k8s.io/kubectl/pkg/cmd/version"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func New(args ...string) *Cmd {
	if len(args) == 0 {
		args = []string{"kubectl"}
	}

	return &Cmd{
		Cmd:    *exec.Command(args[0], args[1:]...), //nolint:gosec
		Logger: slog.Default(),
	}
}

type Cmd struct {
	exec.Cmd
	Logger *slog.Logger
}

func (cmd *Cmd) Server(server string) *Cmd {
	return cmd.SubCmd("--server", server)
}

func (cmd *Cmd) Cluster(cluster string) *Cmd {
	return cmd.SubCmd("--cluster", cluster)
}

func (cmd *Cmd) SubCmd(args ...string) *Cmd {
	return &Cmd{
		Cmd: exec.Cmd{
			Path: cmd.Path,
			Dir:  cmd.Dir,
			Env:  slices.Clone(cmd.Env),
			Args: slices.Concat(cmd.Args, args),
		},
		Logger: cmd.Logger,
	}
}

func (cmd *Cmd) Version() (*version.Version, error) {
	subcmd := cmd.SubCmd("version", "-ojson")
	parser := jsonParser(&version.Version{}, nil)

	return executeCmd(subcmd, parser, nil)
}

func (cmd *Cmd) ClientVersion() (*version.Version, error) {
	subcmd := cmd.SubCmd("version", "-ojson", "--client=true")
	parser := jsonParser(&version.Version{}, nil)

	return executeCmd(subcmd, parser, nil)
}

func (cmd *Cmd) ApplyKustomization(path string) error {
	subcmd := cmd.SubCmd("apply", "--kustomize", path)
	_, err := executeCmd(subcmd, parseNoop, true)

	return err
}

func (cmd *Cmd) BuildKustomization(path string) ([]*yaml.RNode, error) {
	subcmd := cmd.SubCmd(
		"kustomize",
		"--enable-helm=true",
		"--load-restrictor=LoadRestrictionsNone",
		path,
	)

	return executeCmd(subcmd, parseRNodes, nil)
}

func (cmd *Cmd) Get(resources []string, namespace string, selectors []string, names ...string) ([]*yaml.RNode, error) {
	args := []string{"get", "-oyaml"}

	if len(resources) > 0 {
		args = append(args, strings.Join(resources, ","))
	} else {
		args = append(args, "all")
	}

	if len(selectors) > 0 {
		args = append(args, "-l", strings.Join(selectors, ","))
	}

	if namespace != "" {
		args = append(args, "-n", namespace)
	} else {
		args = append(args, "-A")
	}

	args = append(args, names...)
	subcmd := cmd.SubCmd(args...)

	return executeCmd(subcmd, parseRNodes, nil)
}

func (cmd *Cmd) APIResources(namespaced bool) ([]string, error) {
	subcmd := cmd.SubCmd(
		"api-resources",
		"-o", "name",
		"--verbs", "get",
		"--namespaced="+strconv.FormatBool(namespaced),
	)

	resources, err := executeCmd(subcmd, parseLines, nil)
	slices.Sort(resources)

	return resources, err
}

func (cmd *Cmd) Namespaces() ([]string, error) {
	subcmd := cmd.SubCmd(
		"get", "namespaces",
		"-o", "name",
		"--no-headers",
	)

	namespaces, err := executeCmd(subcmd, parseResNames, nil)
	slices.Sort(namespaces)

	return namespaces, err
}

func (cmd *Cmd) Clusters(selectors []types.ClusterSelector) (*Clusters, error) {
	subcmd := cmd.SubCmd("config", "get-clusters")

	lines, err := executeCmd(subcmd, parseLines, nil)
	if err != nil {
		return nil, err
	}

	names := lines[1:]
	slices.Sort(names)

	clusters := &Clusters{
		cmd:          cmd,
		ClusterIndex: types.BuildClusterIndex(names, selectors),
	}

	return clusters, nil
}

func (cmd *Cmd) wrapExecErr(err error) error {
	if err == nil {
		return nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		msg := strings.TrimSpace(string(exitErr.Stderr))

		return fmt.Errorf("failed to execute %q: %w, %s", cmd, err, msg)
	}

	return fmt.Errorf("failed to execute %q: %w", cmd, err)
}

func (cmd *Cmd) wrapParseErr(err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("failed to parse %q output: %w", cmd, err)
}

//nolint:ireturn
func executeCmd[T any](cmd *Cmd, parser parserFn[T], def T) (T, error) {
	data, err := cmd.Output()
	if err == nil {
		result, err := parser(data)

		return result, cmd.wrapParseErr(err)
	}

	return def, cmd.wrapExecErr(err)
}
