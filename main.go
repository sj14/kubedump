package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/exp/slices"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

var (
	// will be replaced during the build process
	version = "undefined"
	commit  = "undefined"
	date    = "undefined"
)

func main() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("failed getting user home dir: %v\n", err)
	}

	var (
		kubeConfigPath    = flag.String("config", filepath.Join(homeDir, ".kube", "config"), "path to the kubeconfig")
		kubeContext       = flag.String("context", "", "context from the kubeconfig, empty for default")
		outdirFlag        = flag.String("dir", "dump", "output directory for the dumps")
		resourcesFlag     = flag.String("resources", "", "resource to dump (e.g. 'configmaps,secrets'), empty for all")
		namespacesFlag    = flag.String("namespaces", "", "namespace to dump (e.g. 'ns1,ns2'), empty for all")
		clusterscopedFlag = flag.Bool("clusterscoped", true, "dump cluster-wide resources")
		namespacedFlag    = flag.Bool("namespaced", true, "dump namespaced resources")
		statelessFlag     = flag.Bool("stateless", true, "remove fields containing a state of the resource")
		versionFlag       = flag.Bool("version", false, fmt.Sprintf("print version information of this release (%v)", version))
	)
	flag.Parse()

	if *versionFlag {
		fmt.Printf("version: %v\n", version)
		fmt.Printf("commit: %v\n", commit)
		fmt.Printf("date: %v\n", date)
		os.Exit(0)
	}

	var (
		wantResources  = strings.Split(strings.ToLower(*resourcesFlag), ",")
		wantNamespaces = strings.Split(strings.ToLower(*namespacesFlag), ",")
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

	k8sClient, err := client.New(kubeConfig, client.Options{})
	if err != nil {
		log.Fatalf("failed creating Kubernetes client: %v\n", err)
	}

	for _, group := range groups.Groups {
		for _, version := range group.Versions {
			resources, err := clientset.DiscoveryClient.ServerResourcesForGroupVersion(version.GroupVersion)
			if err != nil {
				log.Printf("failed getting resources for %q: %v\n", version.GroupVersion, err)
				continue
			}

			for _, res := range resources.APIResources {
				if strings.Contains(res.Name, "/") {
					// skip subresources
					// TODO: probably there is a better way to not get them in the first place
					continue
				}

				// check if we got the specified resources (if any resources were specified)
				if *resourcesFlag != "" && !slices.Contains(wantResources, res.Name) {
					continue
				}

				unstrList := &unstructured.UnstructuredList{}
				unstrList.SetGroupVersionKind(schema.GroupVersionKind{
					Group:   group.Name,
					Version: version.Version,
					Kind:    res.Kind,
				})
				err = k8sClient.List(context.Background(), unstrList)
				if err != nil {
					log.Printf("failed listing %v: %v\n", unstrList.GroupVersionKind().String(), err)
					continue
				}

				for _, listItem := range unstrList.Items {
					// filter according to flags
					if listItem.GetNamespace() != "" && !*namespacedFlag {
						continue
					}
					if listItem.GetNamespace() == "" && !*clusterscopedFlag {
						continue
					}
					if *namespacesFlag != "" && !slices.Contains(wantNamespaces, listItem.GetNamespace()) {
						continue
					}

					namespacedName := fmt.Sprintf("%v/%v", listItem.GetNamespace(), listItem.GetName())

					item := &unstructured.Unstructured{}
					item.SetGroupVersionKind(schema.GroupVersionKind{
						Group:   group.Name,
						Version: version.Version,
						Kind:    res.Kind,
					})

					err := k8sClient.Get(context.Background(), client.ObjectKeyFromObject(&listItem), item)
					if err != nil {
						log.Printf("failed getting %v: %v\n", namespacedName, err)
						continue
					}

					if err := writeYAML(*outdirFlag, res.Name, *item, *statelessFlag); err != nil {
						log.Printf("failed writing %v: %v\n", namespacedName, err)
						continue
					}
				}
			}
		}
	}
}

// TODO: check if we can get the resourceName from the item
func writeYAML(outDir, resourceName string, item unstructured.Unstructured, stateless bool) error {
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

	dir := filepath.Join(outDir, namespace, resourceName)
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
	item.Object = deleteField(item.Object, "metadata", "annotations", "control-plane.alpha.kubernetes.io/leader")
	item.Object = deleteField(item.Object, "metadata", "annotations", "kubectl.kubernetes.io/last-applied-configuration")
	item.Object = deleteField(item.Object, "metadata", "creationTimestamp")
	item.Object = deleteField(item.Object, "metadata", "finalizers")
	item.Object = deleteField(item.Object, "metadata", "generation")
	item.Object = deleteField(item.Object, "metadata", "managedFields")
	item.Object = deleteField(item.Object, "metadata", "resourceVersion")
	item.Object = deleteField(item.Object, "metadata", "selfLink")
	item.Object = deleteField(item.Object, "metadata", "ownerReferences")
	item.Object = deleteField(item.Object, "metadata", "uid")
	item.Object = deleteField(item.Object, "status")

	if item.GetNamespace() == "" {
		// cluster-scoped only
	} else {
		// namespaced only
		item.Object = deleteField(item.Object, "metadata", "annotations", "autoscaling.alpha.kubernetes.io/conditions")
		item.Object = deleteField(item.Object, "metadata", "annotations", "autoscaling.alpha.kubernetes.io/current-metrics")
		item.Object = deleteField(item.Object, "metadata", "annotations", "deployment.kubernetes.io/revision")
		item.Object = deleteField(item.Object, "metadata", "annotations", "kubernetes.io/config.seen")
		item.Object = deleteField(item.Object, "metadata", "annotations", "kubernetes.io/service-account.uid")
		item.Object = deleteField(item.Object, "metadata", "annotations", "pv.kubernetes.io/bind-completed")
		item.Object = deleteField(item.Object, "metadata", "annotations", "pv.kubernetes.io/bound-by-controller")
		item.Object = deleteField(item.Object, "metadata", "clusterIP")
		item.Object = deleteField(item.Object, "metadata", "progressDeadlineSeconds")
		item.Object = deleteField(item.Object, "metadata", "revisionHistoryLimit")
		item.Object = deleteField(item.Object, "metadata", "spec", "metadata", "annotations", "kubectl.kubernetes.io/restartedAt")
		item.Object = deleteField(item.Object, "metadata", "spec", "metadata", "creationTimestamp")
		item.Object = deleteField(item.Object, "spec", "volumeName")
		item.Object = deleteField(item.Object, "spec", "volumeMode")
	}
}

func deleteField(object map[string]interface{}, path ...string) map[string]interface{} {
	if len(path) == 0 {
		return object
	}
	if len(path) == 1 {
		delete(object, path[0])
		return object
	}

	subObj, ok := object[path[0]].(map[string]interface{})
	if !ok {
		return object
	}

	object[path[0]] = deleteField(subObj, path[1:]...)
	return object
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
