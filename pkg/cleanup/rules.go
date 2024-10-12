package cleanup

import (
	"regexp"

	"github.com/Mirantis/rekustomize/pkg/yutil"
	"sigs.k8s.io/kustomize/kyaml/resid"
	kyutil "sigs.k8s.io/kustomize/kyaml/utils"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func DefaultRules() Rules {
	regexpRules := map[string]string{
		"status":                     `.*`,
		"metadata.uid":               `.*`,
		"metadata.selfLink":          `.*`,
		"metadata.resourceVersion":   `.*`,
		"metadata.generation":        `.*`,
		"metadata.creationTimestamp": `.*`,
		"metadata.annotations.[kubectl.kubernetes.io/last-applied-configuration]": `.*`,
		"metadata.annotations.[deployment.kubernetes.io/revision]":                `^Deployment\.v1\.apps/.*$`,
	}
	rules := []Rule{
		&schemaRule{},
		skipRule{
			// TODO: refine rules (e.g. current rules drop non-system ClusterRoles)
			resid.NewResId(resid.NewGvk("", "v1", "ClusterRole"), ""),
			resid.NewResId(resid.NewGvk("", "v1", "ClusterRoleBinding"), ""),
			resid.NewResId(resid.NewGvk("", "v1", "ComponentStatus"), ""),
			resid.NewResId(resid.NewGvk("", "v1", "ConfigMap"), "kube-root-ca.crt"),
			resid.NewResId(resid.NewGvk("", "v1", "Event"), ""),
			resid.NewResId(resid.NewGvk("", "v1", "Namespace"), ""),
			resid.NewResId(resid.NewGvk("", "v1", "Pod"), ""),
			resid.NewResId(resid.NewGvk("", "v1", "ReplicaSet"), ""),
			resid.NewResId(resid.NewGvk("", "v1", "ServiceAccount"), "default"),
			resid.NewResId(resid.NewGvk("admissionregistration.k8s.io", "v1", ""), ""),
			resid.NewResId(resid.NewGvk("apiregistration.k8s.io", "v1", ""), ""),
			resid.NewResId(resid.NewGvk("coordination.k8s.io", "v1", ""), ""),
			resid.NewResId(resid.NewGvk("discovery.k8s.io", "v1", ""), ""),
			resid.NewResId(resid.NewGvk("events.k8s.io", "v1", ""), ""),
			resid.NewResId(resid.NewGvk("flowcontrol.apiserver.k8s.io", "v1", ""), ""),
			resid.NewResId(resid.NewGvk("scheduling.k8s.io", "v1", ""), ""),
			resid.NewResIdWithNamespace(resid.Gvk{}, "", "kube-public"),
			resid.NewResIdWithNamespace(resid.Gvk{}, "", "kube-system"),
			resid.NewResIdWithNamespace(resid.NewGvk("", "v1", "ConfigMap"), "kubernetes", "default"),
			resid.NewResIdWithNamespace(resid.NewGvk("", "v1", "Endpoints"), "kubernetes", "default"),
			resid.NewResIdWithNamespace(resid.NewGvk("", "v1", "Service"), "kubernetes", "default"),
		},
	}
	for pathStr, regexpStr := range regexpRules {
		path := yutil.Path(kyutil.SmarterPathSplitter(pathStr, "."))
		rules = append(rules, &regexpRule{regexp.MustCompile(regexpStr), path})
	}
	return rules
}

type regexpRule struct {
	regexp *regexp.Regexp
	path   yutil.Path
}

func (r *regexpRule) Apply(rn *yaml.RNode) error {
	id := resid.FromRNode(rn).String()
	if !r.regexp.MatchString(id) {
		return nil
	}
	if len(r.path) < 1 {
		return nil
	}
	filters := []yaml.Filter{}
	path, name := r.path[:len(r.path)-1], r.path[len(r.path)-1]
	if len(path) > 0 {
		filters = append(filters, yaml.Lookup(r.path[:len(r.path)-1]...))
	}
	filters = append(filters, yaml.Clear(name))
	rn.Pipe(filters...)
	return nil
}

type Rule interface {
	Apply(*yaml.RNode) error
}

type Rules []Rule

func (r Rules) Filter(nodes []*yaml.RNode) ([]*yaml.RNode, error) {
	result := []*yaml.RNode{}
filterNodes:
	for _, rn := range nodes {
		for _, rule := range r {
			err := rule.Apply(rn)
			if err == skipErr {
				// FIXME: refactor
				continue filterNodes
			}
			if err != nil {
				return nil, err
			}
		}
		result = append(result, rn)
	}
	return result, nil
}
