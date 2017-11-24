package client

import (
	k8s "k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type K8sClient struct {
	clientset *k8s.Clientset
}

func NewK8sClient() *K8sClient {

	config, err := clientcmd.BuildConfigFromFlags("192.168.30.128:8080", "")
	if err != nil {
		panic(err.Error())
	}
	// creates the clientSet
	clientSet, err := k8s.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	client := &K8sClient{
		clientset: clientSet,
	}
	return client
}

func (c *K8sClient) WatchLogs(namespace string, podName string, options *v1.PodLogOptions) *rest.Request {
	return c.clientset.CoreV1Client.Pods(namespace).GetLogs(podName, options)
}
