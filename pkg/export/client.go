package export

import "sigs.k8s.io/kustomize/kyaml/yaml"

// Client is an abstraction layer to support export via kubectl or the native
// Go K8s client library.
type Client interface {
	ApiResources(namespaced bool) ([]string, error)
	Namespaces() ([]string, error)

	Get(kind, namespace string, selectors []string, names ...string) ([]*yaml.RNode, error)
}
