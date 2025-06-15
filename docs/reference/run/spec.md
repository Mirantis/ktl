

<a name="run-proto"></a>




<a name="apis-ClusterSelector"></a>

### ClusterSelector



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| match_names | [PatternSelector](#apis-PatternSelector) | optional |  |
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






<a name="apis-HelmChartOutput"></a>

### HelmChartOutput



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| version | [string](#string) |  |  |






<a name="apis-KubeConfigSource"></a>

### KubeConfigSource



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| path | [string](#string) | optional |  |
| clusters | [ClusterSelector](#apis-ClusterSelector) | repeated |  |
| resources | [ResourceMatcher](#apis-ResourceMatcher) | repeated |  |






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






<a name="apis-MCPToolOutput"></a>

### MCPToolOutput



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| description | [string](#string) |  |  |
| columns | [ColumnOutput](#apis-ColumnOutput) | repeated |  |






<a name="apis-Output"></a>

### Output



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| kustomize | [KustomizeOutput](#apis-KustomizeOutput) | optional |  |
| kustomize_components | [KustomizeComponentsOutput](#apis-KustomizeComponentsOutput) | optional |  |
| helm_chart | [HelmChartOutput](#apis-HelmChartOutput) | optional |  |
| csv | [ColumnarFileOutput](#apis-ColumnarFileOutput) | optional |  |
| table | [ColumnarFileOutput](#apis-ColumnarFileOutput) | optional |  |
| mcp_tool | [MCPToolOutput](#apis-MCPToolOutput) | optional |  |






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
| source | [Source](#apis-Source) |  | Source specifies the origin of the manifests |
| filters | [Filter](#apis-Filter) | repeated | Filters transform the manifests |
| output | [Output](#apis-Output) |  | Output specifies the format of the result |






<a name="apis-ResourceMatcher"></a>

### ResourceMatcher



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| match_names | [PatternSelector](#apis-PatternSelector) | optional |  |
| match_namespaces | [PatternSelector](#apis-PatternSelector) | optional |  |
| match_api_resources | [PatternSelector](#apis-PatternSelector) | optional |  |
| label_selectors | [string](#string) | repeated |  |






<a name="apis-ResourceSelector"></a>

### ResourceSelector



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| group | [string](#string) | optional |  |
| version | [string](#string) | optional |  |
| kind | [string](#string) | optional |  |
| name | [string](#string) | optional |  |
| namespace | [string](#string) | optional |  |
| annotation_selector | [string](#string) | optional |  |
| label_selector | [string](#string) | optional |  |






<a name="apis-SkipFilter"></a>

### SkipFilter



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resources | [ResourceMatcher](#apis-ResourceMatcher) | optional |  |
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

 <!-- end enums -->

 <!-- end HasExtensions -->



