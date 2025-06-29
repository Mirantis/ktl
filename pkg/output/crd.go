package output

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/Mirantis/ktl/pkg/types"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

type CRDDescriptionsOutput struct {
	Path string `yaml:"path"`
}

func (out *CRDDescriptionsOutput) Store(env *types.Env, resources *types.ClusterResources) error {
	result := map[string]string{}

	for resId, byCluster := range resources.Resources {
		for _, rnode := range byCluster {
			var crd apiextensionsv1.CustomResourceDefinition

			ystr, err := rnode.String()
			if err != nil {
				panic(err)
			}

			if err := yaml.Unmarshal([]byte(ystr), &crd); err != nil {
				return fmt.Errorf("unable to process %s: %w", resId, err)
			}

			//TODO: parameterize selection
			slices.SortFunc(
				crd.Spec.Versions,
				func(a, b apiextensionsv1.CustomResourceDefinitionVersion) int {
					return -strings.Compare(a.Name, b.Name)
				},
			)

			if len(crd.Spec.Versions) < 1 {
				continue
			}

			schema := crd.Spec.Versions[0].Schema.OpenAPIV3Schema
			updateSchemaAttrs(result, crd.Spec.Names.Kind+"[]", schema, "")

			//TODO: handle multi-cluster (error or cluster-specific)
			break
		}
	}

	body, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("unable to generate CRD summary: %w", err)
	}

	if err := env.FileSys.WriteFile(out.Path, body); err != nil {
		return fmt.Errorf("unable to store CRD summary: %w", err)
	}

	return nil
}

func updateSchemaAttrs(out map[string]string, prefix string, schema *apiextensionsv1.JSONSchemaProps, extraDescr string) {
	descr := []string{}
	if len(schema.Description) > 0 {
		descr = append(descr, schema.Description)
	}

	switch schema.Type {
	case "object":
		for attr, attrSchema := range schema.Properties {
			extraAttrDescr := ""
			if slices.Contains(schema.Required, attr) {
				extraAttrDescr = "Required."
			}
			updateSchemaAttrs(out, prefix+"."+attr, &attrSchema, extraAttrDescr)
		}
	case "array":
		if schema.Items.Schema != nil {
			updateSchemaAttrs(out, prefix+"[]", schema.Items.Schema, "")
		}
		for _, itemSchema := range schema.Items.JSONSchemas {
			updateSchemaAttrs(out, prefix+"[]", &itemSchema, "")
		}
	default:
		descr = append(descr, "Must be a "+schema.Type+".")
	}

	if len(extraDescr) > 0 {
		descr = append(descr, extraDescr)
	}

	out[prefix] = strings.Join(descr, "\n")
}
