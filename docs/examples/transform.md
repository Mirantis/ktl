# Transform Manifests

[`pipeline.yaml`](../reference/run/spec.md#pipeline) can include multiple
[filters](../reference/run/spec.md#filter) to transform the manifests:

* [`skip`](../reference/run/spec.md#skipfilter) can remove unwanted fields,
  values or entire resources

* [`starlark`](../reference/run/spec.md#starlarkfilter) enables advanced
  transformations via Python-like
  [Starlark-go](https://github.com/google/starlark-go) scripts

The following example uses [kustomize](../reference/run/spec.md#kustomizesource)
source to convert previously generated manifests from Helm Chart to Kustomize
components, and also removing the `sidecar` container with its' ConfigMap - e.g.
the `sidecar` was injected by a controller and should not be included in the
manifests.

```yaml
source:
  kustomize:
    path: '../run-gen-helm/overlays/${CLUSTER}'
    clusters:
    - alias: dev
      matchNames: { include: [ 'dev-*' ] }
    - alias: prod
      matchNames: { include: [ 'prod-*' ] }
filters:
- skip:
    resources:
    - kind: ConfigMap
- skip:
    fields:
    - spec.template.spec.containers[name=sidecar]
output:
  kustomizeComponents: {}
```

As a result only the image version is different between the cluster manifests:

=== "dev"

    ```yaml
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: demo-app
      namespace: ktl-examples
    spec:
      template:
        spec:
          containers:
          - name: demo-app
            image: demo-app:v2
    ```

=== "prod"

    ```yaml
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: demo-app
      namespace: ktl-examples
    spec:
      template:
        spec:
          containers:
          - name: demo-app
            image: demo-app:v1
    ```

And only `dev` and `prod` components are generated:

```
components
├── all-clusters
│   ├── ktl-examples
│   │   ├── demo-app-deployment.yaml
│   │   └── demo-app-service.yaml
│   └── kustomization.yaml
├── dev
│   ├── ktl-examples
│   │   └── demo-app-deployment.yaml
│   └── kustomization.yaml
└── prod
    ├── ktl-examples
    │   └── demo-app-deployment.yaml
    └── kustomization.yaml
```

