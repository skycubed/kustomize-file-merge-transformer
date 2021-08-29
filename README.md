## Kustomize File Merge Transformer Plugin

Kustomize plugin to append the contents of multiple files into a single file. While Kustomize can merge
multiple files into a single ConfigMap, where each file has a separate key (filename), it lacks the ability
to merge the content of multiple keys (file) into a single key (file).

The motivation for this plugin is to generate a single properties file needed by an application for different
environments (dev, test, prod).

## Implementation

This plugin is written in golang, however, it is a kustomize exec plugin, which does not utilize the go
plugin feature. Kustomize exec plugins are called as executables, which can be written in any language.
When executing the plugin, kustomize passes the kubernetes resources yaml to stdin of the executable and
optionally passes arguments in the form of a "one liner" of arguments or a file containing arguments. 
This plugin uses the former. Golang was chosen for the implementation since the kustomize api exposes 
Kubernetes yaml serde functions, which facilitate modification (transform) of Kubernetes resources.

## Example

You must have `kustomize` installed to run the example. If you have go installed, you can install kustomize by running:

```
make install_kustomize
```

The example starts with a base set of properties files contain configuration common to all environments. There are
two properties files, `alpha.properties` and `bravo.properties`. Each of these are configuration files whose 
application requires these file names specifically.

alpha.properties:
```
alpha.base.prop=1
other.prop=2
```

bravo.properties:
```
some.base.prop=1
some.other.base.prop=2
```

For our development environment, we need to append additional properties to each of these files. To do this,
we create additional properties files with names beginning with `alpha` and `bravo`. Multiple files can be
appended to base files. For `alpha` we wish to append two files, so we create the following files: 
`alpha-a.properties` and `alpha-b.properties`. Note that `a` and `b` are not significant. The only requirement is
that the file prefix match the name of the base file (the text before the last `.` and file extension)

The `merge.yaml` file configures the transformer plugin. It's contents are as follows:

```
apiVersion: skycubed.github.com/v1
kind: FileMergeTransformer

metadata:
  name: app-config

argsOneLiner: |
  alpha.properties,bravo.properties
```

The value of `argsOneLiner` tells the plugin that `alpha.properties` and `bravo.properties` are merge targets. This
means that any files within the same ConfigMap with a file name starting with `alpha` or `bravo` will be appended to
their respective targets. Note that the `meatadata.name` property in `merge.yaml` must match the `metadata.name` 
property of the target ConfigMap's `metadata.name` property. If these names do not match, the properties files will
not be merged.

To demonstrate, comment out the following lines in `example/overlays/development/kustomization.yaml`:

```
#transformers:
#  - merge.yaml
```

Then run the following:

```
make example-transform
```

This will disable the plugin and output what native Kustomize generates; a ConfigMap that combines multiple, 
separate files:

```
apiVersion: v1
data:
  alpha-a.properties: a.props=3
  alpha-b.properties: b.props=4
  alpha.properties: |-
    alpha.base.prop=1
    other.prop=2
  bravo-a.properties: bravo.data.here=3
  bravo.properties: |-
    some.base.prop=1
    some.other.base.prop=2
kind: ConfigMap
metadata:
  name: app-config
```

Now, uncomment the lines in `example/overlays/development/kustomization.yaml`, enabling the plugin. 

Run `make example-transform` again to see the desired output:

```
apiVersion: v1
data:
  alpha.properties: |-
    alpha.base.prop=1
    other.prop=2
    a.props=3
    b.props=4
  bravo.properties: |-
    some.base.prop=1
    some.other.base.prop=2
    bravo.data.here=3
kind: ConfigMap
metadata:
  name: app-config
```

Assuming you have a local kubernetes (eg. minikube) running, you can apply the transformed example 
to kubernetes by running:

```
make deploy-example
```

Then verify that files were merged and mounted:

```
kubectl exec $(kubectl get pods | \
  grep myapp | awk '{ print $1 }') -- grep '=' /tmp/{alpha,bravo}.properties
```

## Installation

To install the plugin, run:

```
GOBIN=${XDG_CONFIG_HOME:-~/.config}/kustomize/plugin/skycubed.github.com/v1/filemergetransformer \
  go install github.com/skycubed/kustomize-file-merge-transformer/cmd/FileMergeTransformer@latest
```
