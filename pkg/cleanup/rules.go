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
	rules := []Rule{&schemaRule{}}
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
	for _, rule := range r {
		for _, rn := range nodes {
			if err := rule.Apply(rn); err != nil {
				return nil, err
			}
		}
	}
	return nodes, nil
}
