package export

import "sigs.k8s.io/kustomize/kyaml/yaml"

// Client is an abstraction layer to support export via kubectl or the native
// Go K8s client library.
type Client interface {
	ApiResources() ([]string, error)
	// TODO: change signature to support an equivalent of kubectl's:
	// * --namespace/--all-namespaces
	// * --selector
	Get(kind string) ([]*yaml.RNode, error)
}
