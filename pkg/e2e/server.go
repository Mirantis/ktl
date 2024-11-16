package e2e

import (
	"context"
	"os"
	"testing"
	"time"

	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	ServerVersion = "v1.30.4"
	kwokImage     = "registry.k8s.io/kwok/cluster:v0.6.1-k8s." + ServerVersion
)

func init() {
	// for podman run manually:
	// echo "docker.host: unix://`podman machine inspect \
	// | jq -r '.[0]|.ConnectionInfo.PodmanSocket.Path'`" \
	// > ~/.testcontainers.properties

	os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")
}

func K8sServer(t *testing.T) string {
	const targetPort = "8080"
	ctx := context.Background()
	req := tc.ContainerRequest{
		Image:        kwokImage,
		ExposedPorts: []string{targetPort + "/tcp"},
		WaitingFor:   wait.ForLog("Server Version: " + ServerVersion),
	}

	kwok, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("unable to start K8s (KWOK): %v", err)
	}
	t.Cleanup(func() {
		immediate := time.Second * 0
		if err := kwok.Stop(ctx, &immediate); err != nil {
			t.Errorf("unable to stop K8s (KWOK): %v", err)
		}
		if err := kwok.Terminate(ctx); err != nil {
			t.Errorf("unable to cleanup K8s (KWOK): %v", err)
		}
	})

	port, err := kwok.MappedPort(ctx, targetPort)
	if err != nil {
		t.Fatalf("unable to obtain K8s API port: %v", err)
	}
	return "localhost:" + port.Port()
}
