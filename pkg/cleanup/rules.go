package cleanup

import (
	"regexp"

	"github.com/Mirantis/rekustomize/pkg/yutil"
	"sigs.k8s.io/kustomize/kyaml/resid"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func DefaultRules() Rules {
	return []Rule{
		&regexpRule{regexp.MustCompile(`.*`), yutil.Path{"status"}},
		&regexpRule{regexp.MustCompile(`.*`), yutil.Path{"metadata", "uid"}},
		&regexpRule{regexp.MustCompile(`.*`), yutil.Path{"metadata", "selfLink"}},
		&regexpRule{regexp.MustCompile(`.*`), yutil.Path{"metadata", "resourceVersion"}},
		&regexpRule{regexp.MustCompile(`.*`), yutil.Path{"metadata", "generation"}},
		&regexpRule{regexp.MustCompile(`.*`), yutil.Path{"metadata", "creationTimestamp"}},
		&regexpRule{regexp.MustCompile(`.*`), yutil.Path{"metadata", "annotations", "kubectl.kubernetes.io/last-applied-configuration"}},
		&regexpRule{regexp.MustCompile(`^Deployment\.v1\.apps/.*$`), yutil.Path{
			"metadata", "annotations", "deployment.kubernetes.io/revision",
		}},
	}
}

type regexpRule struct {
	regexp *regexp.Regexp
	path   yutil.Path
}

func (r *regexpRule) Apply(rn *yaml.RNode) {
	id := resid.FromRNode(rn).String()
	if !r.regexp.MatchString(id) {
		return
	}
	if len(r.path) < 1 {
		return
	}
	filters := []yaml.Filter{}
	path, name := r.path[:len(r.path)-1], r.path[len(r.path)-1]
	if len(path) > 0 {
		filters = append(filters, yaml.Lookup(r.path[:len(r.path)-1]...))
	}
	filters = append(filters, yaml.Clear(name))
	rn.Pipe(filters...)
}

type Rule interface {
	Apply(*yaml.RNode)
}

type Rules []Rule

func (r Rules) Filter(nodes []*yaml.RNode) ([]*yaml.RNode, error) {
	for _, rule := range r {
		for _, rn := range nodes {
			rule.Apply(rn)
		}
	}
	return nodes, nil
}
