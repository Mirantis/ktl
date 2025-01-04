package helm

import (
	"fmt"

	"github.com/Mirantis/rekustomize/pkg/yutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func SetOptional(condition string, rn *yaml.RNode, token string, path ...string) error {
	return rn.PipeE(
		yaml.Lookup(path...),
		yutil.CommentsSetter{
			HeadComment: fmt.Sprintf("HELM%s: {{- if index .Values.global \"%s\" }}", token, condition),
			FootComment: fmt.Sprintf("HELM%s: {{- end }}", token),
		},
	)
}

func SetOptionalValue(value string, rn *yaml.RNode, token string, path ...string) error {
	return rn.PipeE(
		yaml.Lookup(path...),
		yaml.Set(yaml.NewScalarRNode("")),
		yutil.CommentsSetter{
			HeadComment: fmt.Sprintf("HELM%s: {{- if index .Values.global \"%s\" }}", token, value),
			LineComment: fmt.Sprintf("HELM%s: {{ index .Values.global \"%s\" }}", token, value),
			FootComment: fmt.Sprintf("HELM%s: {{- end }}", token),
		},
	)
}

func SetValue(value string, rn *yaml.RNode, token string, path ...string) error {
	return rn.PipeE(
		yaml.Lookup(path...),
		yaml.Set(yaml.NewScalarRNode("")),
		yutil.CommentsSetter{
			LineComment: fmt.Sprintf("HELM%s: {{ index .Values.global \"%s\" }}", token, value),
		},
	)
}
