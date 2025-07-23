package kquery

import (
	"sigs.k8s.io/kustomize/kyaml/openapi"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type Node struct {
	parent *Node
	value  *yaml.RNode

	lazySchema *openapi.ResourceSchema
}

type NodeSet struct {
	nodes *Node
}
