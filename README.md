# KTL - K8s manifest generator, transformer, and analyzer

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

👉[Read the Docs](https://mirantis.github.io/ktl/)

