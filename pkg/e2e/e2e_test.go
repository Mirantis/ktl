package e2e_test

import (
	_ "embed"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Mirantis/rekustomize/pkg/cmd"
	"github.com/Mirantis/rekustomize/pkg/e2e"
	"github.com/Mirantis/rekustomize/pkg/kubectl"
	"github.com/Mirantis/rekustomize/pkg/types"
	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

type testServer struct {
	name string
	url  string
	err  error
}

func initServers(t *testing.T, clusters []string) map[string]string {
	t.Helper()

	kctl := kubectl.DefaultCmd()
	result := map[string]string{}

	serversChan := make(chan *testServer)
	defer close(serversChan)

	for _, cluster := range clusters {
		go func(name string) {
			clusterDir := filepath.Join("..", "..", "examples", "import", name)
			url := e2e.K8sServer(t)
			errs := []error{}
			errs = append(errs, kctl.Server(url).ApplyKustomization(clusterDir))
			server := &testServer{name: name, url: url, err: errors.Join(errs...)}
			serversChan <- server
		}(cluster)
	}

	errs := []error{}

	for range clusters {
		entry := <-serversChan
		result[entry.name] = entry.url
		errs = append(errs, entry.err)
	}

	if err := errors.Join(errs...); err != nil {
		t.Fatal(err)
	}

	return result
}

func TestE2E(t *testing.T) {
	testServers := initServers(t, []string{
		"dev-cluster-a",
		"test-cluster-a",
		"test-cluster-b",
		"prod-cluster-a",
		"prod-cluster-b",
	})
	e2e.KubeConfig(t, testServers, "test-cluster-b")

	t.Run("client-version", testClientVersion)
	t.Run("server-version-error", testServerVersionError)
	t.Run("server-version", testServerVersion)

	scenarios := []string{
		"export-simple",
		"export-simple-filtered",
		"export-helm",
		"export-components",
	}

	for _, scenario := range scenarios {
		t.Run(scenario, func(t *testing.T) { testExport(t, scenario) })
	}
}

func testClientVersion(t *testing.T) {
	version, err := kubectl.DefaultCmd().Version(false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if version.ClientVersion.Major != "1" {
		t.Fatalf("unexpected ClientVersion: %+#v", version.ClientVersion)
	}
}

func testServerVersionError(t *testing.T) {
	wantErr := ("kubectl --server 127.0.0.1:1 failed: exit status 1, " +
		"stderr: The connection to the server 127.0.0.1:1 was refused - " +
		"did you specify the right host or port?")

	_, err := kubectl.DefaultCmd().Server("127.0.0.1:1").Version(true)
	if err == nil {
		t.Fatalf("want err, got nil")
	}

	if err.Error() != wantErr {
		t.Fatalf("unexpected error: %v", err)
	}
}

func testServerVersion(t *testing.T) {
	kctl := kubectl.DefaultCmd()

	version, err := kctl.Version(true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if version.ServerVersion.GitVersion != e2e.ServerVersion {
		t.Fatalf("unexpected ServerVersion: %+#v", version.ServerVersion)
	}
}

func testExport(t *testing.T, name string) {
	t.Helper()

	diskFs := filesys.MakeFsOnDisk()
	outDir := filepath.Join(t.TempDir(), name)

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cfg, err := os.ReadFile(filepath.Join("..", "..", "examples", name, types.DefaultFileName))
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(outDir, types.DefaultFileName), cfg, 0o600); err != nil {
		t.Fatal(err)
	}

	exportCmd := cmd.RootCommand()
	exportCmd.SetArgs([]string{"export", outDir})

	if err := exportCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	got := e2e.ReadFiles(t, diskFs, outDir)
	want := e2e.ReadFiles(t, diskFs, filepath.Join("..", "..", "examples", name))

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected result, +got -want:\n%v", diff)
	}
}
