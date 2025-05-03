package e2e_test

import (
	"embed"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Mirantis/rekustomize/pkg/cmd"
	"github.com/Mirantis/rekustomize/pkg/e2e"
	"github.com/Mirantis/rekustomize/pkg/kubectl"
	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

var (
	//go:embed testdata/import/*
	//go:embed testdata/convert-components/pipeline.yaml
	inputConvertComponents embed.FS

	//go:embed testdata/convert-components/*
	wantConvertComponents embed.FS

	//go:embed testdata/import/*
	//go:embed testdata/convert-csv/pipeline.yaml
	inputConvertCSV embed.FS

	//go:embed testdata/convert-csv/*
	wantConvertCSV embed.FS

	//go:embed testdata/import/*
	//go:embed testdata/convert-table/pipeline.yaml
	inputConvertTable embed.FS

	//go:embed testdata/convert-table/*
	wantConvertTable embed.FS

	//go:embed testdata/export-components/pipeline.yaml
	inputExportComponents embed.FS

	//go:embed testdata/export-components/*
	wantExportComponents embed.FS

	//go:embed testdata/export-helm/pipeline.yaml
	inputExportHelm embed.FS

	//go:embed testdata/export-helm/*
	//go:embed testdata/export-helm/charts/simple-app/templates/_helpers.tpl
	wantExportHelm embed.FS

	//go:embed testdata/export-simple/pipeline.yaml
	inputExportSimple embed.FS

	//go:embed testdata/export-simple/*
	wantExportSimple embed.FS

	//go:embed testdata/export-simple-filtered/pipeline.yaml
	inputExportSimpleFiltered embed.FS

	//go:embed testdata/export-simple-filtered/*
	wantExportSimpleFiltered embed.FS
)

type testServer struct {
	name string
	url  string
	err  error
}

func initServers(t *testing.T, clusters []string) map[string]string {
	t.Helper()

	kctl := kubectl.New()
	result := map[string]string{}

	serversChan := make(chan *testServer)
	defer close(serversChan)

	for _, cluster := range clusters {
		go func(name string) {
			clusterDir := filepath.Join("testdata", "import", name)
			url := e2e.K8sServer(t)
			errs := []error{}
			kctl := kctl.Server(url)
			errs = append(errs, kctl.ApplyKustomization(clusterDir))
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

	scenarios := map[string]struct {
		input fs.FS
		want  fs.FS
	}{
		"export-simple": {
			input: inputExportSimple,
			want:  wantExportSimple,
		},
		"export-simple-filtered": {
			input: inputExportSimpleFiltered,
			want:  wantExportSimpleFiltered,
		},
		"export-helm": {
			input: inputExportHelm,
			want:  wantExportHelm,
		},
		"export-components": {
			input: inputExportComponents,
			want:  wantExportComponents,
		},
		"convert-components": {
			input: inputConvertComponents,
			want:  wantConvertComponents,
		},
		"convert-csv": {
			input: inputConvertCSV,
			want:  wantConvertCSV,
		},
		"convert-table": {
			input: inputConvertTable,
			want:  wantConvertTable,
		},
	}

	for name, files := range scenarios {
		t.Run(name, func(t *testing.T) { testRunCmd(t, name, files.input, files.want) })
	}
}

func testClientVersion(t *testing.T) {
	version, err := kubectl.New().ClientVersion()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if version.ClientVersion.Major != "1" {
		t.Fatalf("unexpected ClientVersion: %+#v", version.ClientVersion)
	}
}

func testServerVersionError(t *testing.T) {
	kctl := kubectl.New().Server("127.0.0.1:1")

	_, err := kctl.Version()
	if err == nil {
		t.Fatalf("want err, got nil")
	}

	if !strings.HasPrefix(err.Error(), "failed to execute") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func testServerVersion(t *testing.T) {
	kctl := kubectl.New()

	version, err := kctl.Version()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if version.ServerVersion.GitVersion != e2e.ServerVersion {
		t.Fatalf("unexpected ServerVersion: %+#v", version.ServerVersion)
	}
}

func testRunCmd(t *testing.T, name string, inputFS, wantFS fs.FS) {
	t.Helper()

	var err error

	testDir := t.TempDir()
	diskFs := filesys.MakeFsOnDisk()
	outDir := filepath.Join(testDir, name)

	inputFS, err = fs.Sub(inputFS, "testdata")
	if err != nil {
		t.Fatal(err)
	}

	wantFS, err = fs.Sub(wantFS, "testdata/"+name)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.CopyFS(testDir, inputFS); err != nil {
		t.Fatal(err)
	}

	runCmd := cmd.NewRootCommand()
	runCmd.SetArgs([]string{"run", outDir + "/pipeline.yaml"})

	if err := runCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	got := e2e.ReadFiles(t, diskFs, outDir+"/")
	want := e2e.ReadFsFiles(t, wantFS, ".")

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected result, +got -want:\n%v", diff)
	}
}
