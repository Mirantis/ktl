package e2e_test

import (
	_ "embed"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/Mirantis/rekustomize/pkg/cmd"
	"github.com/Mirantis/rekustomize/pkg/e2e"
	"github.com/Mirantis/rekustomize/pkg/kubectl"
	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/resid"
)

func e2eSubtest(kctl kubectl.Cmd, test func(*testing.T, kubectl.Cmd)) func(*testing.T) {
	return func(t *testing.T) {
		test(t, kctl)
	}
}

func TestE2E(t *testing.T) {
	testServers := map[string]string{
		"server-a": e2e.K8sServer(t),
	}
	e2e.KubeConfig(t, testServers, "server-a")
	kctl := kubectl.DefaultCmd()
	err := kctl.ApplyKustomization("../testdata/server-a")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("client-version", testClientVersion)
	t.Run("server-version-error", testServerVersionError)
	t.Run("server-version", e2eSubtest(kctl, testServerVersion))
	t.Run("get-deployments", e2eSubtest(kctl, testGetDeployments))
	t.Run("export", e2eSubtest(kctl, testExport))
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

func testServerVersion(t *testing.T, kctl kubectl.Cmd) {
	v, err := kctl.Version(true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.ServerVersion.GitVersion != e2e.ServerVersion {
		t.Fatalf("unexpected ServerVersion: %+#v", v.ServerVersion)
	}
}

func testGetDeployments(t *testing.T, kctl kubectl.Cmd) {
	want := []string{
		"Deployment.v1.apps/nginx-a.default",
		"Deployment.v1.apps/nginx-b.default",
	}

	resources, err := kctl.Get("deployments")
	if err != nil {
		t.Fatal(err)
	}
	got := []string{}
	for _, resource := range resources {
		got = append(got, resid.FromRNode(resource).String())
	}
	sort.Strings(got)

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected IDs, +got -want:\n%v", diff)
	}
}

func testExport(t *testing.T, kctl kubectl.Cmd) {
	diskFs := filesys.MakeFsOnDisk()
	outDir := filepath.Join(t.TempDir(), "export")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}
	exportCmd := cmd.RootCommand()
	exportCmd.SetArgs([]string{"export", outDir})

	if err := exportCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	got := e2e.ReadFiles(diskFs, outDir)
	want := e2e.ReadFiles(diskFs, "testdata/export")
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected result, +got -want:\n%v", diff)
	}
}
