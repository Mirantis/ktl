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
	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

type testServer struct {
	name string
	url  string
}

func initServers(t *testing.T, clusters []string) map[string]string {
	kctl := kubectl.DefaultCmd()
	result := map[string]string{}
	ch := make(chan *testServer)
	errs := []error{}
	defer close(ch)
	for _, cluster := range clusters {
		go func(name string) {
			url := e2e.K8sServer(t)
			err := kctl.Server(url).ApplyKustomization("testdata/import/" + name)
			if err != nil {
				errs = append(errs, err)
			}
			ch <- &testServer{name: name, url: url}
		}(cluster)
	}
	for range clusters {
		entry := <-ch
		result[entry.name] = entry.url
	}
	if len(errs) > 0 {
		t.Fatal(errors.Join(errs...))
	}
	return result
}

func TestE2E(t *testing.T) {
	testServers := initServers(t, []string{
		"cluster-a",
		"cluster-b",
		"cluster-c",
		"cluster-d",
		"cluster-e",
	})
	e2e.KubeConfig(t, testServers, "cluster-a")

	t.Run("client-version", testClientVersion)
	t.Run("server-version-error", testServerVersionError)
	t.Run("server-version", testServerVersion)
	t.Run("export", testExport)
	t.Run("export-multi-cluster", testExportMultiCluster)
}

func testClientVersion(t *testing.T) {
	v, err := kubectl.DefaultCmd().Version(false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.ClientVersion.Major != "1" {
		t.Fatalf("unexpected ClientVersion: %+#v", v.ClientVersion)
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
	v, err := kctl.Version(true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.ServerVersion.GitVersion != e2e.ServerVersion {
		t.Fatalf("unexpected ServerVersion: %+#v", v.ServerVersion)
	}
}

func testExport(t *testing.T) {
	diskFs := filesys.MakeFsOnDisk()
	outDir := filepath.Join(t.TempDir(), "export")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}
	exportCmd := cmd.RootCommand()
	exportCmd.SetArgs([]string{
		"export",
		"--namespaces", "default",
		"--resources", "!namespaces",
		outDir})

	if err := exportCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	got := e2e.ReadFiles(diskFs, outDir)
	want := e2e.ReadFiles(diskFs, "testdata/export")
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected result, +got -want:\n%v", diff)
	}
}

func testExportMultiCluster(t *testing.T) {
	diskFs := filesys.MakeFsOnDisk()
	outDir := filepath.Join(t.TempDir(), "export-multi")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}
	exportCmd := cmd.RootCommand()
	exportCmd.SetArgs([]string{
		"export",
		"--clusters", "prod=cluster-a,cluster-b,dev=cluster-c,stage=cluster-[de]",
		"--namespaces", "default",
		"--resources", "!namespaces",
		outDir})

	if err := exportCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	got := e2e.ReadFiles(diskFs, outDir)
	want := e2e.ReadFiles(diskFs, "testdata/export-multi")
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected result, +got -want:\n%v", diff)
	}
}
