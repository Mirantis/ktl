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
	err  error
}

func initServers(t *testing.T, clusters []string) map[string]string {
	kctl := kubectl.DefaultCmd()
	result := map[string]string{}
	ch := make(chan *testServer)
	defer close(ch)
	for _, cluster := range clusters {
		go func(name string) {
			url := e2e.K8sServer(t)
			errs := []error{}
			errs = append(errs, kctl.Server(url).ApplyKustomization("testdata/import/common"))
			errs = append(errs, kctl.Server(url).ApplyKustomization("testdata/import/"+name))
			server := &testServer{name: name, url: url, err: errors.Join(errs...)}
			ch <- server
		}(cluster)
	}
	errs := []error{}
	for range clusters {
		entry := <-ch
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
		"--clusters", "dev-cluster-a",
		"--namespaces", "my*app",
		"-R", "namespaces,customresourcedefinitions.apiextensions.k8s.io",
		"-l", "skip-me!=yes",
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
		"--clusters", "dev=dev-*,test=test-cluster-[ab],prod=prod-cluster-a,prod-cluster-b",
		"--namespaces", "my*app",
		"-R", "namespaces,customresourcedefinitions.apiextensions.k8s.io",
		"-l", "skip-me!=yes",
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
