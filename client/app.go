package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"

	k8client "github.com/byted/client-go-example/client/internal"
	"k8s.io/client-go/util/homedir"
)

func main() {
	kubeconfig := flag.String("kubeconfig-path", filepath.Join(homedir.HomeDir(), ".kube", "config"), "(optional) Path to kubeconfig")
	newNamespaceName := flag.String("new-ns-name", "my-new-namespace", "(optional) Name of the namespace to be created")
	newPodName := flag.String("new-pod-name", "my-new-pod", "(optional) Name of the pod to be created")
	labelSelector := flag.String("label-selector", "k8s-app=kube-dns", "(optional) Label selector to filter pods by")
	createOnly := flag.Bool("createOnly", false, "(optional) Only execute create resource operations")
	deleteOnly := flag.Bool("deleteOnly", false, "(optional) Only execute delete resource operations")
	flag.Parse()

	client, err := k8client.New(*kubeconfig)
	if err != nil {
		log.Fatal(err)
	}

	listResources(client, *labelSelector)
	fmt.Println()

	if !*deleteOnly {
		createResources(client, *newNamespaceName, *newPodName)
		if !*createOnly {
			fmt.Println("All resources created. Press [Enter] to continue and clean up the cluster")
			fmt.Scanln()
		}
	}

	if !*createOnly {
		deleteResources(client, *newNamespaceName, *newPodName)
	}

}

func listResources(client k8client.K8client, labelSelector string) {
	namespaces, err := client.FetchNamespaces()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("available namespaces:", namespaces)

	nsToPods, err := client.ListPodsByLabel(labelSelector)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("available pods with label", labelSelector, ":", nsToPods)
}

func createResources(client k8client.K8client, newNamespaceName string, newPodName string) {
	err := client.CreateNamespace(newNamespaceName)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("created namespace:", newNamespaceName)

	err = client.CreatePod(newNamespaceName, newPodName)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("created pod", newPodName, "in namespace", newNamespaceName)

	port, err := client.ExposePodOnNode(newNamespaceName, newPodName, 30000)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("exposed pod", newPodName, "on node port", port)
}

func deleteResources(client k8client.K8client, newNamespaceName string, newPodName string) {
	err := client.DeletePod(newNamespaceName, newPodName)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("deleted pod", newPodName, "in namespace", newNamespaceName)

	err = client.DeleteNamespace(newNamespaceName)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("deleted namespace", newNamespaceName)
}
