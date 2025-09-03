package e2e_test

import (
	"bytes"
	"embed"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Mirantis/ktl/pkg/cmd"
	"github.com/Mirantis/ktl/pkg/e2e"
	_ "github.com/Mirantis/ktl/pkg/filters" // register filters
	"github.com/Mirantis/ktl/pkg/kubectl"
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
	//go:embed testdata/convert-json/pipeline.yaml
	inputConvertJSON embed.FS

	//go:embed testdata/convert-json/*
	wantConvertJSON embed.FS

	//go:embed testdata/import/*
	//go:embed testdata/convert-starlark/pipeline.yaml
	inputConvertStarlark embed.FS

	//go:embed testdata/convert-starlark/pipeline.yaml
	wantConvertStarlark embed.FS

	//go:embed testdata/convert-starlark/stdout.txt
	wantConvertStarlarkStdout string

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

	//go:embed all:testdata/export-helm/*
	wantExportHelm embed.FS

	//go:embed testdata/export-simple/pipeline.yaml
	inputExportSimple embed.FS

	//go:embed testdata/export-simple/*
	wantExportSimple embed.FS

	//go:embed testdata/export-simple-filtered/pipeline.yaml
	inputExportSimpleFiltered embed.FS

	//go:embed testdata/export-simple-filtered/*
	wantExportSimpleFiltered embed.FS

	//go:embed testdata/import/*
	//go:embed testdata/describe-crds/pipeline.yaml
	inputDescribeCRDs embed.FS

	//go:embed testdata/describe-crds/pipeline.yaml
	wantDescribeCRDs embed.FS

	//go:embed testdata/describe-crds/stdout.json
	wantDescribeCRDsStdout string

	//go:embed testdata/query/output.txt
	wantQueryOutput string
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
		dir        string
		args       []string
		input      fs.FS
		want       fs.FS
		wantStdout string
	}{
		"export-simple": {
			dir:   "export-simple",
			args:  []string{"run"},
			input: inputExportSimple,
			want:  wantExportSimple,
		},
		"export-simple-filtered": {
			dir:   "export-simple-filtered",
			args:  []string{"run"},
			input: inputExportSimpleFiltered,
			want:  wantExportSimpleFiltered,
		},
		"export-helm": {
			dir:   "export-helm",
			args:  []string{"run"},
			input: inputExportHelm,
			want:  wantExportHelm,
		},
		"export-components": {
			dir:   "export-components",
			args:  []string{"run"},
			input: inputExportComponents,
			want:  wantExportComponents,
		},
		"convert-components": {
			dir:   "convert-components",
			args:  []string{"run"},
			input: inputConvertComponents,
			want:  wantConvertComponents,
		},
		"convert-csv": {
			dir:   "convert-csv",
			args:  []string{"run"},
			input: inputConvertCSV,
			want:  wantConvertCSV,
		},
		"convert-json": {
			dir:   "convert-json",
			args:  []string{"run"},
			input: inputConvertJSON,
			want:  wantConvertJSON,
		},
		"convert-starlark": {
			dir:        "convert-starlark",
			args:       []string{"run"},
			input:      inputConvertStarlark,
			want:       wantConvertStarlark,
			wantStdout: wantConvertStarlarkStdout,
		},
		"convert-table": {
			dir:   "convert-table",
			args:  []string{"run"},
			input: inputConvertTable,
			want:  wantConvertTable,
		},
		"describe-crds": {
			dir:        "describe-crds",
			args:       []string{"run"},
			input:      inputDescribeCRDs,
			want:       wantDescribeCRDs,
			wantStdout: wantDescribeCRDsStdout,
		},
	}

	for name, test := range scenarios {
		t.Run(name, func(t *testing.T) {
			testPipelineCmd(
				t,
				test.dir,
				test.args,
				test.input,
				test.want,
				test.wantStdout,
			)
		})
	}

	t.Run("test-query", testQuery)
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

func testPipelineCmd(t *testing.T, dir string, args []string, inputFS, wantFS fs.FS, wantOut string) {
	t.Helper()

	var err error

	testDir := t.TempDir()
	diskFs := filesys.MakeFsOnDisk()
	outDir := filepath.Join(testDir, dir)

	inputFS, err = fs.Sub(inputFS, "testdata")
	if err != nil {
		t.Fatal(err)
	}

	wantFS, err = fs.Sub(wantFS, "testdata/"+dir)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.CopyFS(testDir, inputFS); err != nil {
		t.Fatal(err)
	}

	gotOut := bytes.NewBuffer(nil)
	runCmd := cmd.NewRootCommand()
	runCmd.SetArgs(append(args, outDir+"/pipeline.yaml"))
	runCmd.SetOut(gotOut)

	if err := runCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	got := e2e.ReadFiles(t, diskFs, outDir+"/")
	want := e2e.ReadFsFiles(t, wantFS, ".")

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected result, +got -want:\n%v", diff)
	}

	if diff := cmp.Diff(wantOut, gotOut.String()); diff != "" {
		t.Errorf("unexpected stdout, +got -want:\n%v", diff)
	}
}

func testQuery(t *testing.T) {
	gotOut := bytes.NewBuffer(nil)
	runCmd := cmd.NewRootCommand()
	runCmd.SetArgs([]string{
		"query",
		"-n", "simple-app",
		"-C", "CONTAINER:spec.template.spec.containers.*.name,IMAGE:spec.template.spec.containers.*.image",
		"*",
		`it.kind == "Service"`,
		"or",
		//FIXME: test with unset nodes
		`it.kind in ["Deployment", "ReplicaSet"]`,
		`and`,
		`it.metadata.name.startswith("simple-app-db")`,
	})
	runCmd.SetOut(gotOut)

	if err := runCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(wantQueryOutput, gotOut.String()); diff != "" {
		t.Errorf("unexpected stdout, +got -want:\n%v", diff)
	}
}
