

<a name="run-proto"></a>




<a name="apis-Args"></a>

### Args



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| schema | [google.protobuf.Struct](#google-protobuf-Struct) | optional |  |
| schemaFile | [string](#string) | optional |  |






<a name="apis-CRDDescriptionsOutput"></a>

### CRDDescriptionsOutput



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| path | [string](#string) | optional |  |






<a name="apis-ClusterSelector"></a>

### ClusterSelector



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| matchNames | [PatternSelector](#apis-PatternSelector) | optional |  |
| alias | [string](#string) | optional |  |






<a name="apis-ColumnOutput"></a>

### ColumnOutput



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| description | [string](#string) | optional |  |
| field | [string](#string) | optional |  |
| text | [string](#string) | optional |  |






<a name="apis-ColumnarFileOutput"></a>

### ColumnarFileOutput



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| path | [string](#string) | optional |  |
| columns | [ColumnOutput](#apis-ColumnOutput) | repeated |  |






<a name="apis-Filter"></a>

### Filter



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| skip | [SkipFilter](#apis-SkipFilter) | optional |  |
| starlark | [StarlarkFilter](#apis-StarlarkFilter) | optional |  |
| defaults | [DefaultsFilter](#apis-DefaultsFilter) | optional |  |






<a name="apis-HelmChartOutput"></a>

### HelmChartOutput



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| version | [string](#string) |  |  |
| valuesAliases | [HelmChartOutput.ValuesAliasesEntry](#apis-HelmChartOutput-ValuesAliasesEntry) | repeated |  |






<a name="apis-HelmChartOutput-ValuesAliasesEntry"></a>

### HelmChartOutput.ValuesAliasesEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="apis-JSONOutput"></a>

### JSONOutput



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| path | [string](#string) | optional |  |
| schema | [google.protobuf.Struct](#google-protobuf-Struct) | optional |  |






<a name="apis-KubeConfigSource"></a>

### KubeConfigSource



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| path | [string](#string) | optional |  |
| clusters | [ClusterSelector](#apis-ClusterSelector) | repeated |  |
| resources | [ResourceMatcher](#apis-ResourceMatcher) | repeated |  |






<a name="apis-KubectlOutput"></a>

### KubectlOutput



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| kubeconfig | [string](#string) | optional |  |
| cluster | [string](#string) | optional |  |






<a name="apis-KustomizeComponentsOutput"></a>

### KustomizeComponentsOutput







<a name="apis-KustomizeOutput"></a>

### KustomizeOutput







<a name="apis-KustomizeSource"></a>

### KustomizeSource



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| path | [string](#string) |  |  |
| clusters | [ClusterSelector](#apis-ClusterSelector) | repeated |  |






<a name="apis-Output"></a>

### Output



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| kustomize | [KustomizeOutput](#apis-KustomizeOutput) | optional |  |
| kustomizeComponents | [KustomizeComponentsOutput](#apis-KustomizeComponentsOutput) | optional |  |
| helmChart | [HelmChartOutput](#apis-HelmChartOutput) | optional |  |
| csv | [ColumnarFileOutput](#apis-ColumnarFileOutput) | optional |  |
| table | [ColumnarFileOutput](#apis-ColumnarFileOutput) | optional |  |
| crdDescriptions | [CRDDescriptionsOutput](#apis-CRDDescriptionsOutput) | optional |  |
| kubectl | [KubectlOutput](#apis-KubectlOutput) | optional |  |
| json | [JSONOutput](#apis-JSONOutput) | optional |  |






<a name="apis-PatternSelector"></a>

### PatternSelector



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| include | [string](#string) | repeated |  |
| exclude | [string](#string) | repeated |  |






<a name="apis-Pipeline"></a>

### Pipeline
Pipeline defines the combination of source, filters and output.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Name of the pipeline |
| description | [string](#string) |  | Description for the pipeline |
| source | [Source](#apis-Source) |  | Source specifies the origin of the manifests |
| filters | [Filter](#apis-Filter) | repeated | Filters transform the manifests |
| output | [Output](#apis-Output) |  | Output specifies the format of the result |
| args | [Args](#apis-Args) | optional | Args describe pipeline parameters |






<a name="apis-ResourceMatcher"></a>

### ResourceMatcher



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| matchNames | [PatternSelector](#apis-PatternSelector) | optional |  |
| matchNamespaces | [PatternSelector](#apis-PatternSelector) | optional |  |
| matchApiResources | [PatternSelector](#apis-PatternSelector) | optional |  |
| labelSelectors | [string](#string) | repeated |  |






<a name="apis-ResourceSelector"></a>

### ResourceSelector



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| group | [string](#string) | optional |  |
| version | [string](#string) | optional |  |
| kind | [string](#string) | optional |  |
| name | [string](#string) | optional |  |
| namespace | [string](#string) | optional |  |
| annotationSelector | [string](#string) | optional |  |
| labelSelector | [string](#string) | optional |  |






<a name="apis-SkipFilter"></a>

### SkipFilter



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resources | [ResourceSelector](#apis-ResourceSelector) | repeated |  |
| keepResources | [ResourceSelector](#apis-ResourceSelector) | repeated |  |
| fields | [string](#string) | repeated |  |






<a name="apis-Source"></a>

### Source



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| kubeconfig | [KubeConfigSource](#apis-KubeConfigSource) | optional |  |
| kustomize | [KustomizeSource](#apis-KustomizeSource) | optional |  |






<a name="apis-StarlarkFilter"></a>

### StarlarkFilter



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| script | [string](#string) |  |  |





 <!-- end messages -->


<a name="apis-DefaultsFilter"></a>

### DefaultsFilter


| Name | Number | Description |
| ---- | ------ | ----------- |
| UNKNOWN | 0 |  |
| NONE | 1 |  |


 <!-- end enums -->

 <!-- end HasExtensions -->



