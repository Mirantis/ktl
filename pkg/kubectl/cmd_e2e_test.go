package kubectl_test

import (
	_ "embed"
	"io/fs"
	"sort"
	"testing"

	"github.com/Mirantis/rekustomize/pkg/cleanup"
	"github.com/Mirantis/rekustomize/pkg/e2e"
	"github.com/Mirantis/rekustomize/pkg/kubectl"
	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/resid"
)

var (
	//go:embed testdata/export/nginx-a.yaml
	testdataNginxA string
	//go:embed testdata/export/nginx-b.yaml
	testdataNginxB string
)

func e2eSubtest(kctl kubectl.Cmd, test func(*testing.T, kubectl.Cmd)) func(*testing.T) {
	return func(t *testing.T) {
		test(t, kctl)
	}
}

func TestE2E(t *testing.T) {
	server := e2e.K8sServer(t)
	kctl := kubectl.DefaultCmd().Server(server)
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
	wantErr := ("kubectl failed: exit status 1, " +
		"stderr: The connection to the server localhost:8080 was refused - " +
		"did you specify the right host or port?")
	_, err := kubectl.DefaultCmd().Version(true)
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
	resources, err := kctl.Get("deployments")
	if err != nil {
		t.Fatal(err)
	}
	for _, rn := range resources {
		cleanup.DefaultRules().Apply(rn)
		rn.Pipe()
	}
	memfs := filesys.MakeFsInMemory()
	pkg := kio.LocalPackageWriter{
		Kind:        "Kustomization",
		PackagePath: "/",
	}
	pkg.FileSystem.Set(memfs)
	if err := pkg.Write(resources); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"/default/deployment_nginx-a.yaml": testdataNginxA,
		"/default/deployment_nginx-b.yaml": testdataNginxB,
	}
	got := transformKFS(memfs)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected result, +got -want:\n%v", diff)
	}
}

func transformKFS(f filesys.FileSystem) map[string]string {
	got := map[string]string{}
	f.Walk("/", func(path string, info fs.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		data, err := f.ReadFile(path)
		if err != nil {
			return err
		}
		got[path] = string(data)
		return nil
	})
	return got
}
