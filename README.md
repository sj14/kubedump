# kubedump

Tool for dumping manifests from your Kubernetes clusters.

## Installation

### Precompiled Binaries

Binaries are available for all major platforms. See the [releases](https://github.com/sj14/kubedump/releases) page.

### Homebrew

Using the [Homebrew](https://brew.sh/) package manager for macOS:

``` text
brew install sj14/tap/kubedump
```

### Manually

It's also possible to install via `go get`:

``` text
go get -u github.com/sj14/kubedump
```

## Usage

```text
Usage of kubedump:
  -clusterscoped
        dump cluster-wide resources (default true)
  -config string
        path to the kubeconfig (default "~/.kube/config")
  -context string
        context from the kubeconfig, empty for default
  -dir string
        output directory for the dumps (default "dump")
  -namespaced
        dump namespaced resources (default true)
  -namespaces string
        namespace to dump (e.g. 'ns1,ns2'), empty for all
  -resources string
        resource to dump (e.g. 'configmaps,secrets'), empty for all
  -stateless
        remove fields containing a state of the resource (default true)
  -verbose
        output the current progress
  -version
        print version information of this release
```
