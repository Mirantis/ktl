package output_test

import (
	_ "embed"
)

var (
	//go:embed testdata/dev-cluster-a.yaml
	appDevA string
	//go:embed testdata/test-cluster-a.yaml
	appTestA string
	//go:embed testdata/test-cluster-b.yaml
	appTestB string
	//go:embed testdata/prod-cluster-a.yaml
	appProdA string
	//go:embed testdata/prod-cluster-b.yaml
	appProdB string
)
