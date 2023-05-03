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

### In-Cluster

See [deploy/cronjob.yaml](./deploy/cronjob.yaml) as an example how to deploy a CronJob with kubedump.
You have to adjust the file accordingly, for example to push the dumped data to a persistent storage.

## Usage

```text
Usage of kubedump:
  -clusterscoped
        dump cluster-wide resources (default true)
  -config string
        path to the kubeconfig, empty for in-cluster config (default "~/.kube/config")
  -context string
        context from the kubeconfig, empty for default
  -dir string
        output directory for the dumps (default "dump")
  -ignore-namespaces string
        namespace to ignore (e.g. 'ns1,ns2')
  -ignore-resources string
        resource to ignore (e.g. 'configmaps,secrets')
  -namespaced
        dump namespaced resources (default true)
  -namespaces string
        namespace to dump (e.g. 'ns1,ns2'), empty for all
  -resources string
        resource to dump (e.g. 'configmaps,secrets'), empty for all
  -stateless
        remove fields containing a state of the resource (default true)
  -threads uint
        maximum number of threads (minimum 1) (default 10)
  -verbosity uint
        verbosity of the output (0-3) (default 1)
  -version
        print version information of this release
```

All options can also be set as environment variables by using their uppercase flag names and changing dashes (`-`) with underscores (`_`), e.g. `ignore-namespaces` becomes `IGNORE_NAMESPACES`.
