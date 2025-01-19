Each of the `export-*` directoreis contain examples of running `rekustomize
export` for a certain scenario defined by the corresponding
`rekustomization.yaml`. The `rekustomization.yaml` files inside each folder
contain comments describing each scenario and parameters.

The contents of this directory also drives the [e2e tests](
https://github.com/Mirantis/rekustomize/blob/main/pkg/e2e/e2e_test.go)
and is expected to accurately represent the current state of `rekustomize`.

Reproducing the results
-----------------------
1. Install `rekustomize` by running `go install .` at the root of this
repository. If you do not have `$HOME/go/bin` in your `$PATH`, you can copy
the binary (`$HOME/go/bin/rekustomize`) to your preferred location.

2. To ensure your K8s setup is not affected, please use a separete `KUBECONFIG`
for testing `rekustomize`. Make sure to run this command in every shell session
where you expect to test `rekustomize`:
    ```
    export KUBECONFIG=$HOME/.kube/rekustomize_examples_config
    ```

3. The contents of this folder is generated for [KWOK](https://kwok.sigs.k8s.io)
clusters and the resources from the corresponding `import` directory. To
reproduce this setup, run the `prepare-clusters.sh` script. The script requires
a working Docker or Podman CLI. Use `stop-clusters.sh` to cleanup.

4. You can now run `rekustomize` for existing examples, e.g.:
    ```
    rekustomize export export-single
    rekustomize export export-multi
    rekustomize export export-helm
    ```

5. If you would like to experiment with a different `rekustomization.yaml`,
    create a new empty directory with the desired `rekustomization.yaml`.
    Then you can run `rekustomize`:
    ```
    rekustomize export your/custom/export-directory
    ```
