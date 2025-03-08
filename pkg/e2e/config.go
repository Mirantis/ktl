package e2e

import (
	"path/filepath"
	"testing"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func KubeConfig(t *testing.T, clusters map[string]string, current string) {
	t.Helper()

	testDir := t.TempDir()
	cfgFile := filepath.Join(testDir, "kubeconfig")
	t.Setenv(clientcmd.RecommendedConfigPathEnvVar, cfgFile)
	cfgPath := clientcmd.NewDefaultPathOptions()
	cfgPath.GlobalFile = cfgFile

	cfg, err := cfgPath.GetStartingConfig()
	if err != nil {
		t.Fatal(err)
	}

	for name, server := range clusters {
		cfg.Clusters[name] = &api.Cluster{Server: server}
		cfg.Contexts[name] = &api.Context{Cluster: name, Namespace: "default"}
	}

	cfg.CurrentContext = current
	if err := clientcmd.ModifyConfig(cfgPath, *cfg, true); err != nil {
		t.Fatal(err)
	}
}
