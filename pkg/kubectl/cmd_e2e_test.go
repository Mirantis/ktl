package kubectl_test

import (
	"testing"

	"github.com/Mirantis/rekustomize/pkg/e2e"
	"github.com/Mirantis/rekustomize/pkg/kubectl"
)

func TestE2EClientVersion(t *testing.T) {
	v, err := kubectl.DefaultCmd().Version(false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.ClientVersion.Major != "1" {
		t.Fatalf("unexpected ClientVersion: %+#v", v.ClientVersion)
	}
}

func TestE2EServerVersionError(t *testing.T) {
	wantErr := ("kubectl failed: exit status 1, " +
		"stderr: The connection to the server localhost:8080 was refused - " +
		"did you specify the right host or port?")
	_, err := kubectl.DefaultCmd().Version(true)
	if err.Error() != wantErr {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestE2EServerVersion(t *testing.T) {
	server := e2e.K8sServer(t)
	v, err := kubectl.DefaultCmd().Server(server).Version(true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.ServerVersion.GitVersion != e2e.ServerVersion {
		t.Fatalf("unexpected ServerVersion: %+#v", v.ServerVersion)
	}
}
