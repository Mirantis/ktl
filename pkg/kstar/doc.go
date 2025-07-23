// Package kstar provides Starlark interface to operate on K8s resources.
//
// # Built-ins
//
// In addition to standard Starlark built-ins, kstar provides the following
// functions and global variables.
//
// ## `schema`
//
// Can be used to find or create corresponding objects. It supports both
// dict-like and object-like interface, e.g:
//
//		schema["io.k8s.api.core.v1.Container"]
//		schema.io.k8s.api.core.v1.Container`
//
// For convenience, `schema` also supports short suffix-matching form,
// as long as it's unambiguous, e.g.:
//
//		schema.Container
//		schema.core.v1.Container
//
// If short form match exists for both `io.k8s` and CRDs, the latter is
// ignored. Otherwise using the short form when multiple matches exist will
// cause an error.
//
// ## `match`
//
// Creates a shell-like pattern for matching, e.g.:
//
//		"mystring" == match("my*")
//		match("my*", "mystring")
//		match("my*", ["mystring", "other"])
//		["mystring", "other"] | match("my*")
//
// ## `regex`
//
// Creates a regex pattern for matching, similar to `match`.
//
// # Resource operations
//
// TODO: add descriptions
//
//	resources(kind="Pod", metadata_name=match("my*")).metadata.name
//	resources(lambda r: r.kind == "Pod" and r.metadata.name == match("my*"))
//
//	resources.metadata.labels["example.com/my-label"] = "new-label"
//	resources[schema.Container(name="myapp")].image = "new-image"
//
//	resources.spec.containers += schema.Container(
//		name="sidecar",
//		image="mysidecar",
//	)
//
//	resources -= schema.Container(image="deprecated-container")
//
//	resources.metadata.labels += {
//		"new-label1": "value1",
//		"new-label2": "value2",
//	}
package kstar
