# About

`ktl` is a versatile CLI tool for working with Kubernetes manifests. It can pull
resource definitions from live clusters or static files, apply various
transformations, and output results in various formats like *Kustomize*, *Helm*,
*CSV* and more.

Think of it as your Swiss Army knife for:
- Migrating live cluster resources to deployment & *GitOps* tools like
  `k0rdent`, `Flux`, `ArgoCD`, and others
- Normalizing and converting manifests
- Querying and analyzing resources across multiple clusters
- Creating Model Context Protocol (MCP) tools to enrich LLM context

## Usage Scenarios

### Migration & manifest generation tool

Convert your live cluster state into reproducible *Kustomize* packages or *Helm*
charts. Perfect for streamlining migration to services like `k0rdent`, `Flux` or
`ArgoCD`.

```bash
$ ktl run generate.yaml
```

Example of `generate.yaml` to generate a *Kustomize* components package with all
resources from the `simple-app` namespace, except for specific resources:

```yaml
source:
  kind: KubeConfig
  clusters:
  - names: dev-*
    tags: dev
  - names: test-cluster-a,test-cluster-b
    tags: test
  - names:
    - 'prod-*'
    - '*-prod-*'
    tags: prod
  resources:
  - namespaces: simple-app
filters:
- kind: SkipFilter
  resources:
  - kind: CronJob
    name: infra-canary
output:
  kind: KustomizeComponents
```

If you prefer *Helm* chart, just change the `output`:

```yaml
output:
  kind: HelmChart
  helmChart:
    name: simple-app
    version: v1.0
```

### Manifest cleanup, conversion & optimization tool

```bash
$ ktl run convert.yaml
```

Normalize and optimize your existing manifests while also removing unwanted
fields:

```yaml
source:
  kind: Kustomize
  kustomization: "path-to/${CLUSTER}"
  clusters:
  - names: dev-*
    tags: dev
  - names: test-cluster-a,test-cluster-b
    tags: test
  - names:
    - prod-cluster-a
    - prod-cluster-b
    tags: prod
filters:
- kind: SkipFilter
  fields:
  - metadata.annotations.example\.com/generated
  - metadata.annotations.dev\.example\.com/info[=flaky-tests]
output:
  kind: KustomizeComponents
```

Or convert to a *Helm* chart with `output`:

```yaml
output:
  kind: HelmChart
  helmChart:
    name: converted-app
    version: v1.0
```

### Query and analysis tool

Use the `query` command to find matching resources:

```bash
$ ktl query 'deployments.apps,daemonsets' \
  --clusters 'dev-*' \
  'it["spec.template.spec.containers.[image=db:.*]"]'
```

Or use the `run` command for more advanced filtering using the
[Starlark](https://github.com/bazelbuild/starlark) Python dialect:

```bash
$ ktl run query.yaml
```

`query.yaml`:

```yaml
source:
  kind: KubeConfig
  clusters:
  - names: dev-*
    tags: dev
  - names: test-cluster-a,test-cluster-b
    tags: test
  - names:
    - 'prod-*'
    - '*-prod-*'
    tags: prod
  resources:
  - namespaces: simple-app
filters:
- kind: Starlark
  script: |
    for it in resources:
      if not it["spec.template.spec.containers.[image=db:.*]"]:
        continue
      if "test" in str(it.metadata.name):
        continue
      output.append(res)
output:
  kind: Table
  columns:
  - name: CLUSTER
    text: "${CLUSTER}"
  - name: KIND
    field: kind
  - name: NAMESPACE
    field: metadata.namespace
  - name: NAME
    field: metadata.name
  - name: CONTAINER
    field: spec.template.spec.containers.*.name
  - name: IMAGE
    field: spec.template.spec.containers.*.image
```

### MCP tool (proof-of-concept)

Create MCP tools that can provide your LLM with information about resources on
your clusters. Combine with the above `query` filters for advanced scenarios.

*NOTE*: currently the MCP server is implemented as `mcp-bridge-poc.py` script, but
the future versions will provide it as part of the `ktl` binary.

```yaml
source:
  clusters:
  - names: "*"
  resources:
  - apiResources: deployments.apps
output:
  kind: MCPTool
  description: |-
    List containers and their images for
    all deployments in all known K8s clusters
  columns:
  - name: CLUSTER
    text: "${CLUSTER}"
    description: Cluster name
  - name: NAMESPACE
    field: metadata.namespace
    description: Namespace name
  - name: NAME
    field: metadata.name
    description: Deployment name
  - name: CONTAINER
    field: spec.template.spec.containers.*.name
    description: Container name
  - name: IMAGE
    field: spec.template.spec.containers.*.image
    description: Container image
```
