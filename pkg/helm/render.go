package helm

import (
	"strings"

	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func String(token string, rn *yaml.RNode) (string, error) {
	text, err := rn.String()
	if err != nil {
		return "", err
	}
	ret := strings.ReplaceAll(text, "# HELM"+token+": ", "")
	return ret, nil
}
