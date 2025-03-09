# rekustomize (Technical Preview / Aplha)

`rekustomize` lets you generate
[`kustomize`](https://github.com/kubernetes-sigs/kustomize) manifests. With
`rekustomize` you can:

- [x] **Export** resource manifests from live cluster using supported formats:
  - [x] **kustomize components**
    ([example](examples/export-components/rekustomization.yaml))
  - [x] **helm charts** ([example](examples/export-helm/rekustomization.yaml))
- [x] **Filter** exported resources or attributes based on customizable rules
  ([example](examples/export-simple-filtered/rekustomization.yaml))
- [x] **Deduplicate** manifests to ensure
  [DRY](https://en.wikipedia.org/wiki/Don%27t_repeat_yourself) principle
- [ ] **Re-format** existing manifests to follow customizable styles and to
  optimize duplicated manifests
- [ ] **Convert** existing manifests between formats, e.g. from **helm chart**
  to **kustomize components**

## Deduplication of resources and attributes

`rekustomize` truly shines when it's used to generate manifests for multiple
clusters at once. It will group correlad resources together and will also
efficiently handle multiple versions of the same resource.

### Helm template

This [template](examples/export-helm/charts/simple-app/templates/simple-app-deployment.yaml)
shows how `rekustomize` can handle multiple versions of the same resource:

```
spec:
  {{- if index .Values.global "simple-app/Deployment/simple-app.spec.replicas" }}
  replicas: {{ index .Values.global "simple-app/Deployment/simple-app.spec.replicas" }}
  {{- end }} # simple-app/Deployment/simple-app.spec.replicas
  template:
    spec:
      containers:
      - image: {{ index .Values.global "simple-app/Deployment/simple-app.spec.template.spec.containers.[name=simple-app].image" }}
        name: simple-app
```

[values.yaml](examples/export-helm/charts/simple-app/values.yaml)
defines presets to group values from different resources:

```
preset_values:
  prod:
    simple-app/ConfigMap/simple-app-env.data.ENV_VAR2: prod-value
    simple-app/Deployment/simple-app.spec.replicas: 5
  prod_test:
    simple-app/Deployment/simple-app.spec.template.spec.containers.[name=simple-app].image: example.com/simple-app:v1.2.345
  test:
    simple-app/ConfigMap/simple-app-env.data.ENV_VAR2: test-value
    simple-app/Deployment/simple-app.spec.replicas: 3
```

The generated **presets** are then assigned to each corresponding cluster:
- [test-cluster-a](examples/export-helm/overlays/test-cluster-a/kustomization.yaml):

```
  valuesInline:
    presets:
    - prod_test
    - test
```

- [prod-cluster-b](examples/export-helm/overlays/prod-cluster-b/kustomization.yaml):

```
  valuesInline:
    presets:
    - prod
    - prod_test
```

Values, that are unique to a cluster are also supported:
- [prod-cluster-a](examples/export-helm/overlays/prod-cluster-a/kustomization.yaml):
```
  valuesInline:
    global:
      simple-app/ConfigMap/simple-app-env.data.ENV_VAR3: prod-cluster-a-value
```

### Kustomize components

If you use **kustomize components** format, similar overrides will be generated
as strategic-merge patches:

- [components/prod](examples/export-components/components/prod/simple-app/simple-app-deployment.yaml):

```
apiVersion: apps/v1
kind: Deployment
metadata:
  name: simple-app
  namespace: simple-app
spec:
  replicas: 5
```

- [components/test](examples/export-components/components/test/simple-app/simple-app-deployment.yaml):

```
apiVersion: apps/v1
kind: Deployment
metadata:
  name: simple-app
  namespace: simple-app
spec:
  replicas: 3
```

- [components/prod_test](examples/export-components/components/prod_test/simple-app/simple-app-deployment.yaml)

```
apiVersion: apps/v1
kind: Deployment
metadata:
  name: simple-app
  namespace: simple-app
spec:
  template:
    spec:
      containers:
      - name: simple-app
        image: example.com/simple-app:v1.2.345
```

## `rekustomization.yaml`

To start using `rekustomize`, create a new folder with a `rekustomization.yaml`
file where you can configure `rekustomize` behavior. Here's a breakdown of the
**helm chart** [example](examples/export-helm/rekustomization.yaml):

### Cluster selector

By default `rekustomize` will use clusters defined in your `KUBECONFIG` file.
You can select clusters using shell-like patterns:

```
clusters:
- names: dev-* # shell-like pattern
  tags: dev
- names: test-cluster-a,test-cluster-* # comma-separated list
  tags: test
- names: # yaml list
  - prod-cluster-a
  - prod-cluster-*
  tags: prod
```

In the future versions `rekustomize` will be able to use cluster metadata/labels
from external sources, such as **ClusterAPI**, **k0rdent**, **ArgoCD** or
**Flux**.

### Resource selector

By default, `rekustomize` will generate manifests for all resources from all
namespaces, excluding a [pre-defined](pkg/cmd/defaults.yaml) list of dynamic
resources, like `pods` or `events`. You can customize that behavior in the
`export` section:

```
export:
- namespaces: simple-app
  labelSelectors: ['!generated-by']
  apiResources:
    exclude: jobs.batch
- apiResources: namespaces
  names: simple-app
```

### Resource attribute cleanup

If your live cluster resources contain attributes that you don't want to be
included in the generated manifests, you can use `SkipFilter` to remove such
unwanted attributes or values. The following example will remove any
`'example.com/generated'` annotations and `'dev.example.com/info'` annotations
with the value of `'flaky-tests'`:

```
filters:
- kind: SkipFilter
  fields:
  - metadata.annotations.example\.com/generated
  - metadata.annotations.dev\.example\.com/info[=flaky-tests]
```

### Resource filters

You can also use `SkipFilter` to exclude entire resources from the generated
manifests - to further refine [resource selection](#resource-selector):

```
filters:
- kind: SkipFilter
  resources:
  - kind: CronJob
    name: infra-canary
```

### Chart metadata

This section defines metadata for the generated chart:

```
helmChart:
  name: simple-app
  version: v1.0
```

If no `helmChart` is specified, `rekustomize` will generate manifests in **kustomize components** format
