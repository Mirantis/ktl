package examples

import (
	_ "embed"

	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var (
	//go:embed import/dev-cluster-a/myapp/myapp.yaml
	MyAppDeploymentDevAYaml string
	MyAppDeploymentDevA     = yaml.MustParse(MyAppDeploymentDevAYaml)

	//go:embed import/prod-cluster-a/myapp/myapp.yaml
	MyAppDeploymentProdAYaml string
	MyAppDeploymentProdA     = yaml.MustParse(MyAppDeploymentProdAYaml)

	//go:embed import/prod-cluster-b/myapp/myapp.yaml
	MyAppDeploymentProdBYaml string
	MyAppDeploymentProdB     = yaml.MustParse(MyAppDeploymentProdBYaml)

	//go:embed import/test-cluster-a/myapp/myapp.yaml
	MyAppDeploymentTestAYaml string
	MyAppDeploymentTestA     = yaml.MustParse(MyAppDeploymentTestAYaml)

	//go:embed import/test-cluster-b/myapp/myapp.yaml
	MyAppDeploymentTestBYaml string
	MyAppDeploymentTestB     = yaml.MustParse(MyAppDeploymentTestBYaml)

	//go:embed export-helm/charts/myapp/templates/myapp-deployment.yaml
	MyAppDeploymentTemplate string
)
