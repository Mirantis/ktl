package kubectl_test

import (
	"testing"

	"github.com/Mirantis/rekustomize/pkg/kubectl"
)

func TestE2EClientVersion(t *testing.T) {
	v, err := kubectl.DefaultCmd.Version(false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.ClientVersion.Major != "1" {
		t.Fatalf("unexpected ClientVersion.Major: %v", v.ClientVersion.Major)
	}
}

func TestE2EServerVersion(t *testing.T) {
	wantErr := ("kubectl failed: exit status 1, " +
		"stderr: The connection to the server localhost:8080 was refused - " +
		"did you specify the right host or port?")
	_, err := kubectl.DefaultCmd.Version(true)
	if err.Error() != wantErr {
		t.Fatalf("unexpected error: %v", err)
	}
}
