package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/exp/slices"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/yaml"
)

var (
	// will be replaced during the build process
	version = "undefined"
	commit  = "undefined"
	date    = "undefined"
)

func main() {
	start := time.Now()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("failed getting user home dir: %v\n", err)
	}

	var (
		kubeConfigPath       = flag.String("config", filepath.Join(homeDir, ".kube", "config"), "path to the kubeconfig")
		kubeContext          = flag.String("context", "", "context from the kubeconfig, empty for default")
		outdirFlag           = flag.String("dir", "dump", "output directory for the dumps")
		resourcesFlag        = flag.String("resources", "", "resource to dump (e.g. 'configmaps,secrets'), empty for all")
		ignoreResourcesFlag  = flag.String("ignore-resources", "", "resource to ignore (e.g. 'configmaps,secrets')")
		namespacesFlag       = flag.String("namespaces", "", "namespace to dump (e.g. 'ns1,ns2'), empty for all")
		ignoreNamespacesFlag = flag.String("ignore-namespaces", "", "namespace to ignore (e.g. 'ns1,ns2')")
		clusterscopedFlag    = flag.Bool("clusterscoped", true, "dump cluster-wide resources")
		namespacedFlag       = flag.Bool("namespaced", true, "dump namespaced resources")
		statelessFlag        = flag.Bool("stateless", true, "remove fields containing a state of the resource")
		verboseFlag          = flag.Bool("verbose", false, "output the current progress")
		versionFlag          = flag.Bool("version", false, fmt.Sprintf("print version information of this release (%v)", version))
	)
	flag.Parse()

	if *versionFlag {
		fmt.Printf("version: %v\n", version)
		fmt.Printf("commit: %v\n", commit)
		fmt.Printf("date: %v\n", date)
		os.Exit(0)
	}

	var (
		wantResources    = strings.Split(strings.ToLower(*resourcesFlag), ",")
		wantNamespaces   = strings.Split(strings.ToLower(*namespacesFlag), ",")
		ignoreResources  = strings.Split(strings.ToLower(*ignoreResourcesFlag), ",")
		ignoreNamespaces = strings.Split(strings.ToLower(*ignoreNamespacesFlag), ",")
	)

	kubeConfig, err := buildConfigFromFlags(*kubeContext, *kubeConfigPath)
	if err != nil {
		log.Fatalf("failed getting Kubernetes config: %v\n", err)
	}

	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		log.Fatalf("failed getting Kubernetes clientset: %v\n", err)
	}

	groups, err := clientset.DiscoveryClient.ServerGroups()
	if err != nil {
		log.Fatalf("failed getting server groups: %v\n", err)
	}

	dynamicClient, err := dynamic.NewForConfig(kubeConfig)
	if err != nil {
		log.Fatalf("failed creating dynamic client: %v\n", err)
	}

	written := 0
	for _, group := range groups.Groups {
		for _, version := range group.Versions {
			resources, err := clientset.DiscoveryClient.ServerResourcesForGroupVersion(version.GroupVersion)
			if err != nil {
				log.Printf("failed getting resources for %q: %v\n", version.GroupVersion, err)
				continue
			}

			for _, res := range resources.APIResources {
				if skipResource(res, wantResources, ignoreResources) {
					continue
				}

				gvr := schema.GroupVersionResource{
					Group:    group.Name,
					Version:  version.Version,
					Resource: res.Name,
				}

				if *verboseFlag {
					fmt.Printf("processing: %s\n", gvr.String())
				}

				unstrList, err := dynamicClient.Resource(gvr).List(context.Background(), metav1.ListOptions{})
				if err != nil {
					log.Printf("failed listing %v: %v\n", gvr.String(), err)
					continue
				}

				for _, item := range unstrList.Items {
					if skipItem(item, *namespacedFlag, *clusterscopedFlag, wantNamespaces, ignoreNamespaces) {
						continue
					}

					// Use a combination of resource and group name as it might not be unique otherwise.
					// Example content of the variables:
					//		resource: "pod"		group: ""
					//		resource: "pod"		group: "metrics.k8s.io"
					resourceAndGroup := strings.TrimSuffix(fmt.Sprintf("%s.%s", res.Name, group.Name), ".")

					if err := writeYAML(*outdirFlag, resourceAndGroup, item, *statelessFlag); err != nil {
						log.Printf("failed writing %v/%v: %v\n", item.GetNamespace(), item.GetName(), err)
						continue
					}
					written++
				}
			}
		}
	}
	fmt.Printf("loaded %d manifests in %v\n", written, time.Since(start).Round(1*time.Millisecond))
}

func skipResource(res metav1.APIResource, wantResources, ignoreResources []string) bool {
	// check if we can even 'get' the resource
	if !slices.Contains(res.Verbs, "get") {
		return true
	}

	// skip subresources
	// TODO: maybe there is a better way to not get them in the first place
	if strings.Contains(res.Name, "/") {
		return true
	}

	// check if we got the specified resources (if any resources were specified)
	if len(wantResources) > 0 && wantResources[0] != "" && !slices.Contains(wantResources, res.Name) {
		return true
	}

	// check if we got a resource to ignore (if any resources were specified)
	if len(ignoreResources) > 0 && ignoreResources[0] != "" && slices.Contains(ignoreResources, res.Name) {
		return true
	}

	return false
}

func skipItem(item unstructured.Unstructured, namespaced, clusterscoped bool, wantNamespaces, ignoreNamespaces []string) bool {
	// item with namespace but we skip namespaced items
	if item.GetNamespace() != "" && !namespaced {
		return true
	}
	// item clusterscoped but we skip them
	if item.GetNamespace() == "" && !clusterscoped {
		return true
	}
	// specific namespaces specied but doesn't match
	if len(wantNamespaces) > 0 && wantNamespaces[0] != "" && !slices.Contains(wantNamespaces, item.GetNamespace()) {
		return true
	}
	// ignore specific namespaces and it matches
	if len(ignoreNamespaces) > 0 && ignoreNamespaces[0] != "" && slices.Contains(ignoreNamespaces, item.GetNamespace()) {
		return true
	}

	return false
}

func writeYAML(outDir, resourceAndGroup string, item unstructured.Unstructured, stateless bool) error {
	if stateless {
		cleanState(item)
	}

	yamlBytes, err := yaml.Marshal(item.Object)
	if err != nil {
		return fmt.Errorf("failed marshalling: %v", err)
	}

	namespace := "clusterscoped"
	if item.GetNamespace() != "" {
		namespace = filepath.Join("namespaced", item.GetNamespace())
	}

	dir := filepath.Join(outDir, namespace, resourceAndGroup)
	if err = os.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("failed creating dir %q: %v", dir, err)
	}

	objName := strings.ReplaceAll(item.GetName(), ":", "_") // windows compatibility
	filename := filepath.Join(dir, objName) + ".yaml"
	if err = os.WriteFile(filename, yamlBytes, os.ModePerm); err != nil {
		return fmt.Errorf("failed writing file %q: %v", filename, err)
	}

	return nil
}

func cleanState(item unstructured.Unstructured) {
	// partially based on https://github.com/WoozyMasta/kube-dump/blob/f1ae560a8b9da8dba1c28619f38089d40d0d2357/kube-dump#L334

	// cluster-scoped and namespaced
	unstructured.RemoveNestedField(item.Object, "metadata", "annotations", "control-plane.alpha.kubernetes.io/leader")
	unstructured.RemoveNestedField(item.Object, "metadata", "annotations", "kubectl.kubernetes.io/last-applied-configuration")
	unstructured.RemoveNestedField(item.Object, "metadata", "creationTimestamp")
	unstructured.RemoveNestedField(item.Object, "metadata", "finalizers")
	unstructured.RemoveNestedField(item.Object, "metadata", "generation")
	unstructured.RemoveNestedField(item.Object, "metadata", "managedFields")
	unstructured.RemoveNestedField(item.Object, "metadata", "resourceVersion")
	unstructured.RemoveNestedField(item.Object, "metadata", "selfLink")
	unstructured.RemoveNestedField(item.Object, "metadata", "ownerReferences")
	unstructured.RemoveNestedField(item.Object, "metadata", "uid")
	unstructured.RemoveNestedField(item.Object, "status")

	if item.GetNamespace() == "" {
		// cluster-scoped only
	} else {
		// namespaced only
		unstructured.RemoveNestedField(item.Object, "metadata", "annotations", "autoscaling.alpha.kubernetes.io/conditions")
		unstructured.RemoveNestedField(item.Object, "metadata", "annotations", "autoscaling.alpha.kubernetes.io/current-metrics")
		unstructured.RemoveNestedField(item.Object, "metadata", "annotations", "deployment.kubernetes.io/revision")
		unstructured.RemoveNestedField(item.Object, "metadata", "annotations", "kubernetes.io/config.seen")
		unstructured.RemoveNestedField(item.Object, "metadata", "annotations", "kubernetes.io/service-account.uid")
		unstructured.RemoveNestedField(item.Object, "metadata", "annotations", "pv.kubernetes.io/bind-completed")
		unstructured.RemoveNestedField(item.Object, "metadata", "annotations", "pv.kubernetes.io/bound-by-controller")
		unstructured.RemoveNestedField(item.Object, "metadata", "clusterIP")
		unstructured.RemoveNestedField(item.Object, "metadata", "progressDeadlineSeconds")
		unstructured.RemoveNestedField(item.Object, "metadata", "revisionHistoryLimit")
		unstructured.RemoveNestedField(item.Object, "metadata", "spec", "metadata", "annotations", "kubectl.kubernetes.io/restartedAt")
		unstructured.RemoveNestedField(item.Object, "metadata", "spec", "metadata", "creationTimestamp")
		unstructured.RemoveNestedField(item.Object, "spec", "volumeName")
		unstructured.RemoveNestedField(item.Object, "spec", "volumeMode")
	}
}

// https://github.com/kubernetes/client-go/issues/192#issuecomment-349564767
func buildConfigFromFlags(context, kubeconfigPath string) (*rest.Config, error) {
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		&clientcmd.ConfigOverrides{
			CurrentContext: context,
		}).ClientConfig()
	if err != nil {
		return config, err
	}

	// https://kubernetes.io/blog/2020/09/03/warnings/#customize-client-handling
	config = rest.CopyConfig(config)
	config.WarningHandler = rest.NoWarnings{}
	config.QPS = 100
	config.Burst = 300
	return config, nil
}
