# Reports & Queries

`ktl` can be also used to generate reports and queries via the `csv` and `table`
[`output`](../reference/run/spec.md#output) options, combined with the
`starlark` filter.

## List resources

```bash
❯ ktl query pods,deployments.apps \
  -C 'CONTAINER:.spec.template.spec.containers.*.name' \
  -C 'IMAGE:.spec.template.spec.containers.*.image' \
  -n ktl-examples

CLUSTER   KIND         NAMESPACE      NAME                        CONTAINER   IMAGE
dev-a     Deployment   ktl-examples   demo-app                    demo-app    demo-app:v2
dev-a     Deployment   ktl-examples   demo-app                    sidecar     sidecar:v1
dev-a     Pod          ktl-examples   demo-app-85f587d6cf-bcfdv
dev-b     Deployment   ktl-examples   demo-app                    demo-app    demo-app:v2
dev-b     Pod          ktl-examples   demo-app-7f9d96987c-564q6
prod-a    Deployment   ktl-examples   demo-app                    demo-app    demo-app:v1
prod-a    Deployment   ktl-examples   demo-app                    sidecar     sidecar:v1
prod-a    Pod          ktl-examples   demo-app-7d469d5fbb-t2zpq
prod-b    Deployment   ktl-examples   demo-app                    demo-app    demo-app:v1
prod-b    Pod          ktl-examples   demo-app-59c695bc9b-k8d7w
```

## Query via Starlark

```bash
❯ ktl query deployments.apps -n ktl-examples \
  'it["spec.template.spec.containers.[name=sidecar]"]'

CLUSTER   KIND         NAMESPACE      NAME
dev-a     Deployment   ktl-examples   demo-app
prod-a    Deployment   ktl-examples   demo-app
```

## Advanced queries and computed fields

```yaml title="examples/run-report-starlark/pipeline.yaml"
source:
  kustomize:
    path: '../setup/${CLUSTER}'
    clusters:
    - matchNames: { include: [ '*-*' ] }
filters:
- starlark:
    script: |-
      for it in resources:
        if not str(it["metadata.name"]).startswith("kwok"):
          continue
        it.metadata.labels.new = "for-%s" % (it.metadata.name) 
        output.append(it)
output:
  csv:
    path: output.csv
    columns:
    - name: CLUSTER
      text: '${CLUSTER}'
    - name: KIND
      field: .kind
    - name: NAME
      field: .metadata.name
    - name: NEW_LABEL
      field: .metadata.labels.new
```

```csv title="examples/run-report-starlark/output.csv"
CLUSTER,KIND,NAME,NEW_LABEL
dev-a,Node,kwok-node-0,for-kwok-node-0
dev-b,Node,kwok-node-0,for-kwok-node-0
prod-a,Node,kwok-node-0,for-kwok-node-0
prod-b,Node,kwok-node-0,for-kwok-node-0
```

