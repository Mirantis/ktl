# Getting Started

## Installation

1. Currently `ktl` can only be installed from source:
  ``` bash
  brew install go

  go install github.com/Mirantis/ktl
  ```

2. Make sure you have `${HOME}/go/bin` in your `${PATH}`

3. To use `ktl` you will also need `kubectl` and optionally `helm`:
  ``` bash
  brew install kubernetes-cli helm
  ```

## Try the examples

If you would like to try `ktl` in an isolated sandbox environment instead of
your live clusters, please follow these steps:

1. [Download](https://github.com/Mirantis/ktl/archive/refs/heads/main.zip) or
   [clone](https://github.com/mirantis/ktl) the `ktl` repository

2. Run the included `examples/setup.sh` - this will deploy several
   lightweight K8s [`kwok`](https://kwok.sigs.k8s.io/) clusters as Docker/Podman
   containers and will populate them with sample applications

3. Run the `export KUBECONFIG` command returned by `setup.sh` - this will make
   sure `ktl` uses the sample clusters instead of your real clusters from the
   main `KUBECONFIG`

4. To cleanup, run `examples/cleanup.sh`

