package k8client

import (
	"context"
	"fmt"
	"time"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

type K8client interface {
	FetchNamespaces() ([]string, error)
	CreateNamespace(string) error
	DeleteNamespace(string) error
	ListPodsByLabel(string) (map[string][]string, error)
	CreatePod(string, string) error
	DeletePod(string, string) error
	ExposePodOnNode(string, string, int32) (int32, error)
	InitPodExposerInformer()
}

type goClientFacade struct {
	clientset *kubernetes.Clientset
	client    v1.CoreV1Interface
}

func New(kubeconfigPath string) (K8client, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("Creating new K8client failed: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("Creating new K8client failed: %w", err)
	}

	return goClientFacade{clientset, clientset.CoreV1()}, nil
}

func (t goClientFacade) FetchNamespaces() ([]string, error) {
	namespaces, err := t.client.Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("FetchNamespaces failed: %w", err)
	}

	var nsNames []string
	for _, ns := range namespaces.Items {
		nsNames = append(nsNames, ns.Name)
	}
	return nsNames, nil
}

func (t goClientFacade) CreateNamespace(name string) error {
	newNamespace := &apiv1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
	_, err := t.client.Namespaces().Create(context.TODO(), newNamespace, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("CreateNamespace failed: %w", err)
	}
	return nil
}

func (t goClientFacade) DeleteNamespace(nsName string) error {
	err := t.client.Namespaces().Delete(context.TODO(), nsName, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("DeleteNamespace failed: %w", err)
	}
	return nil
}

func (t goClientFacade) ListPodsByLabel(label string) (map[string][]string, error) {
	podList, err := t.client.Pods(apiv1.NamespaceAll).List(context.TODO(), metav1.ListOptions{LabelSelector: label})
	if err != nil {
		return nil, fmt.Errorf("ListPodsByLabel failed: %w", err)
	}

	nsToPods := make(map[string][]string)
	for _, pod := range podList.Items {
		nsToPods[pod.Namespace] = append(nsToPods[pod.Namespace], pod.Name)
	}
	return nsToPods, nil
}

func (t goClientFacade) CreatePod(nsName string, podName string) error {
	podSpec := &apiv1.Pod{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: podName,
			Labels: map[string]string{
				"created-by": "client-go-example",
				"name":       podName,
			},
		},
		Spec: apiv1.PodSpec{
			Containers: []apiv1.Container{
				{
					Name:  "web",
					Image: "nginx:1.12",
					Ports: []apiv1.ContainerPort{
						{
							Name:          "http",
							Protocol:      apiv1.ProtocolTCP,
							ContainerPort: 80,
						},
					},
				},
			},
		},
	}

	_, err := t.client.Pods(nsName).Create(context.TODO(), podSpec, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("CreatePod failed: %w", err)
	}
	return nil
}

func (t goClientFacade) DeletePod(nsName string, podName string) error {
	err := t.client.Pods(nsName).Delete(context.TODO(), podName, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("DeletePod failed: %w", err)
	}
	return nil
}

func (t goClientFacade) ExposePodOnNode(nsName string, podName string, port int32) (int32, error) {
	serviceSpec := &apiv1.Service{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: podName, // User pod name as service name
			Labels: map[string]string{
				"created-by": "client-go-example",
				"name":       podName,
			},
		},
		Spec: apiv1.ServiceSpec{
			Ports: []apiv1.ServicePort{
				{
					Protocol: apiv1.ProtocolTCP,
					Port:     80,
					NodePort: port,
				},
			},
			Selector: map[string]string{"name": podName},
			Type:     "NodePort",
		},
	}

	svc, err := t.client.Services(nsName).Create(context.TODO(), serviceSpec, metav1.CreateOptions{})
	if err != nil {
		return 0, fmt.Errorf("ExposePodOnNode failed: %w", err)
	}
	return svc.Spec.Ports[0].NodePort, nil
}

func (t goClientFacade) InitPodExposerInformer() {
	informerfactory := informers.NewSharedInformerFactory(t.clientset, 60*time.Second)

	podinformer := informerfactory.Core().V1().Pods()
	podinformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			newPod := obj.(*apiv1.Pod)
			if newPod.Labels["created-by"] == "client-go-example" {
				t.ExposePodOnNode(newPod.Namespace, newPod.Name, 30000)
				fmt.Println("Created service for pod", newPod.Name, "to expose on Node")
			}
		},
		DeleteFunc: func(obj interface{}) {
			pod := obj.(*apiv1.Pod)
			if pod.Labels["created-by"] == "client-go-example" {
				fmt.Println("Deleting service for deleted pod", pod.Name, "not implemented")
			}
		},
	})
	informerfactory.Start(wait.NeverStop)
	informerfactory.WaitForCacheSync(wait.NeverStop)
}
