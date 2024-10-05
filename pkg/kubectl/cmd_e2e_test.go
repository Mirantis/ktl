package kubectl_test

import (
	"sort"
	"testing"

	"github.com/Mirantis/rekustomize/pkg/e2e"
	"github.com/Mirantis/rekustomize/pkg/kubectl"
	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/kustomize/kyaml/resid"
)

func e2eSubtest(kctl kubectl.KubectlCmd, test func(*testing.T, kubectl.KubectlCmd)) func(*testing.T) {
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

func testServerVersion(t *testing.T, kctl kubectl.KubectlCmd) {
	v, err := kctl.Version(true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.ServerVersion.GitVersion != e2e.ServerVersion {
		t.Fatalf("unexpected ServerVersion: %+#v", v.ServerVersion)
	}
}

func testGetDeployments(t *testing.T, kctl kubectl.KubectlCmd) {
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
