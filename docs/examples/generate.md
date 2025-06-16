# Generating Manifests

To demonstrate `ktl` ability to generate K8s manifests from live clusters, we
will use the following setup with 4 clusters (`dev-a`, `dev-b`, `prod-a` and
`prod-b`):

1. All clusters have `demo-app` deployed in `ktl-examples` namespace

2. `dev-*` clusters use container image `demo-app:v2` while `prod-*` clusters
   are running `v1`

3. `*-a` clusters have an additional `sidecar` container

4. `dev-a` cluster uses a configmap for the `sidecar` container

To generate the manifests, we can use `ktl run pipeline.yaml` command. Full
specification of the `pipeline.yaml` is available in the [`run`
spec](../reference/run/spec.md#pipeline) section.

## Kustomize Components

```yaml title="examples/run-gen-components/pipeline.yaml"
source:
  kubeconfig:
    clusters:
    - matchNames: { include: [ '*' ] }
    resources:
    - matchNamespaces: { include: [ 'ktl-examples' ] }
output:
  kustomizeComponents: {}
```

As a result `ktl` will generate the following Kustomize layout:

```
components
├── all-clusters
│   ├── ktl-examples
│   │   ├── demo-app-deployment.yaml
│   │   └── demo-app-service.yaml
│   └── kustomization.yaml
├── dev-a
│   ├── ktl-examples
│   │   ├── demo-app-deployment.yaml
│   │   └── sidecar-env-configmap.yaml
│   └── kustomization.yaml
├── dev-a_dev-b
│   ├── ktl-examples
│   │   └── demo-app-deployment.yaml
│   └── kustomization.yaml
├── dev-a_prod-a
│   ├── ktl-examples
│   │   └── demo-app-deployment.yaml
│   └── kustomization.yaml
└── prod-a_prod-b
    ├── ktl-examples
    │   └── demo-app-deployment.yaml
    └── kustomization.yaml
overlays
├── dev-a
│   └── kustomization.yaml
├── dev-b
│   └── kustomization.yaml
├── prod-a
│   └── kustomization.yaml
└── prod-b
    └── kustomization.yaml
```

Here the `demo-app` for cluster `dev-a` is composed from the `all-clusters` base
and 3 patches:

=== "all-clusters"
    ```yaml
    apiVersion: apps/v1
    kind: Deployment
    metadata:
    labels:
      app: demo-app
    name: demo-app
    namespace: ktl-examples
    spec:
    selector:
      matchLabels:
        app: demo-app
    template:
      metadata:
        labels:
          app: demo-app
      spec:
        containers:
        - name: demo-app
    ```
=== "dev-a_dev-b"
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
=== "dev-a_prod-a"
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
          - image: sidecar:v1
            name: sidecar
    ```
=== "dev-a"
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
          - name: sidecar
            envFrom:
            - configMapRef:
                name: sidecar-env
    ```

## Cluster group aliases and resource selection

[`resources`](../reference/run/spec.md#resourcematcher) attribute can be used to
refine resource selection. All `match*` attributes support `shell`-like
[patterns](https://pkg.go.dev/path#Match).

[`alias`](../reference/run/spec.md#apis-ClusterSelector) attribute can be used
to improve component naming.

```yaml title="examples/run-gen-components/pipeline.yaml"
source:
  kubeconfig:
    clusters:
    - alias: dev
      matchNames: { include: [ 'dev-*' ] }
    - alias: prod
      matchNames: { include: [ 'prod-*' ] }
    - alias: env-a
      matchNames: { include: [ '*-a' ] }
    - alias: env-b
      matchNames: { include: [ '*-b' ] }
    resources:
    - matchNamespaces: { include: [ 'ktl-examples' ] }
      matchApiResources: { exclude: [ 'services' ] }

    - matchApiResources: { include: [ 'namespaces' ] }
      matchNames: { include: [ 'ktl-examples' ] }
output:
  kustomizeComponents: {}

```

As a result, the `ktl-examples` namespace will be exported, but not the
service. And component names will use aliases when possible:

```
components
├── all-clusters
│   ├── ktl-examples
│   │   └── demo-app-deployment.yaml
│   ├── ktl-examples-namespace.yaml
│   └── kustomization.yaml
├── dev
│   ├── ktl-examples
│   │   └── demo-app-deployment.yaml
│   └── kustomization.yaml
├── dev-a
│   ├── ktl-examples
│   │   ├── demo-app-deployment.yaml
│   │   └── sidecar-env-configmap.yaml
│   └── kustomization.yaml
├── env-a
│   ├── ktl-examples
│   │   └── demo-app-deployment.yaml
│   └── kustomization.yaml
└── prod
    ├── ktl-examples
    │   └── demo-app-deployment.yaml
    └── kustomization.yaml
```

## Helm Chart

We can also change the output to a Helm chart:

```yaml title="examples/run-gen-helm/pipeline.yaml"
source:
  kubeconfig:
    clusters:
    - alias: dev
      matchNames: { include: [ 'dev-*' ] }
    - alias: prod
      matchNames: { include: [ 'prod-*' ] }
    - alias: env-a
      matchNames: { include: [ '*-a' ] }
    - alias: env-b
      matchNames: { include: [ '*-b' ] }
    resources:
    - matchNamespaces: { include: [ 'ktl-examples' ] }
output:
  helmChart:
    name: demo-app
    version: v1.0
```

As a result, `ktl` will generate the following:

```
charts
└── demo-app
    ├── Chart.yaml
    ├── templates
    │   ├── _helpers.tpl
    │   ├── demo-app-deployment.yaml
    │   ├── demo-app-service.yaml
    │   └── sidecar-env-configmap.yaml
    └── values.yaml
overlays
├── dev-a
│   └── kustomization.yaml
├── dev-b
│   └── kustomization.yaml
├── prod-a
│   └── kustomization.yaml
└── prod-b
    └── kustomization.yaml
```

The resulting deployment template will have all variable attributes
parameterized:

```yaml title="charts/demo-app/templates/demo-app-deployment.yaml"
{{- include "merge_presets" . -}}
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: demo-app
  name: demo-app
  namespace: ktl-examples
spec:
  selector:
    matchLabels:
      app: demo-app
  template:
    metadata:
      labels:
        app: demo-app
    spec:
      containers:
      {{- if index .Values.global "ktl-examples/Deployment/demo-app.spec.template.spec.containers.[name=sidecar]" }}
      - {{- if index .Values.global "ktl-examples/Deployment/demo-app.spec.template.spec.containers.[name=sidecar].envFrom" }}
        envFrom:
        - configMapRef:
            name: sidecar-env
        {{- end }} # ktl-examples/Deployment/demo-app.spec.template.spec.containers.[name=sidecar].envFrom

        image: sidecar:v1
        name: sidecar
      {{- end }} # ktl-examples/Deployment/demo-app.spec.template.spec.containers.[name=sidecar]
      - image: {{ index .Values.global "ktl-examples/Deployment/demo-app.spec.template.spec.containers.[name=demo-app].image" }}
        name: demo-app
```

Common values will be grouped into presets and bundled within chart's
`values.yaml` - simiar to Kustomize components:

```yaml title="charts/demo-app/values.yaml"
global: {}
preset_values:
  dev:
    ktl-examples/Deployment/demo-app.spec.template.spec.containers.[name=demo-app].image: demo-app:v2
  env-a:
    ktl-examples/Deployment/demo-app.spec.template.spec.containers.[name=sidecar]: enabled
  prod:
    ktl-examples/Deployment/demo-app.spec.template.spec.containers.[name=demo-app].image: demo-app:v1
```

The Kustomize `overlays` will contain per-cluster `valuesInline`:

```yaml title="overlays/dev-a/kustomization.yaml"
kind: Kustomization
helmGlobals:
  chartHome: ../../charts
helmCharts:
- name: demo-app
  version: v1.0
  valuesInline:
    global:
      ktl-examples/ConfigMap/sidecar-env: enabled
      ktl-examples/Deployment/demo-app.spec.template.spec.containers.[name=sidecar].envFrom: enabled
    presets:
    - dev
    - env-a
```

