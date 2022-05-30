package main

import (
	"context"
	"fmt"
	"log"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {

	fmt.Println("1. Connecting to k8s cluster")
	// https://kubernetes.io/docs/tasks/administer-cluster/access-cluster-api/#go-client
	k8sConfig, err := clientcmd.BuildConfigFromFlags("", "/Users/conradhanson/.kube/config2")
	if err != nil {
		panic(err)
	}

	clientSet, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	namespaces, err := clientSet.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		panic(err)
	}

	fmt.Println("2. Current Namespaces")
	for _, ns := range namespaces.Items {
		fmt.Println(ns.Name)
	}

	nsName := "conrad-aerospike"
	newNs := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: nsName,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
	}

	fmt.Printf("3. creating a namespace called '%s'\n", nsName)
	if _, err := clientSet.CoreV1().Namespaces().Create(ctx, newNs, metav1.CreateOptions{}); err != nil {
		// panic(err)
		log.Default().Print(err)
	}

	fmt.Printf("4. Creating a pod running a hello-world container in the '%s' namespace\n", nsName)
	helloworldPod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hello-world",
			Namespace: nsName,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "hello-world",
					Image:   "docker.io/busybox:latest",
					Command: []string{"bin/sh"},
					Args:    []string{"-c", "while true; do echo hello world! `date`; sleep 10; done"},
				},
			},
		},
	}
	if _, err := clientSet.CoreV1().Pods(nsName).Create(ctx, helloworldPod, metav1.CreateOptions{}); err != nil {
		panic(err)
	}

	fmt.Println("5. print out pod names and the namespace they are in for any pods that have a label of 'k8s-app=kube-dns'")
	fmt.Println("formatting as namespace/pod-name")
	for _, ns := range namespaces.Items {
		if pods, err := clientSet.CoreV1().Pods(ns.Name).List(ctx, metav1.ListOptions{LabelSelector: "k8s-app=kube-dns"}); err != nil {
			panic(err)
		} else {
			for _, pod := range pods.Items {
				fmt.Printf("-> %s/%s\n", ns.Name, pod.Name)
			}
		}
	}

	fmt.Println("sleeping for 30 seconds")
	time.Sleep(time.Duration(30000000000))

	fmt.Println("6. delete the hello-world pod created from above")
	clientSet.CoreV1().Pods(nsName).Delete(ctx, helloworldPod.Name, metav1.DeleteOptions{})

	// informerExample(clientSet)

	fmt.Println("7. extra credit - show how a client-go informer works")
	informerFactory := informers.NewSharedInformerFactory(clientSet, 0)

	fmt.Println("using an informer for v1/secrets")
	secretInformer := informerFactory.Core().V1().Secrets()

	stopCh := make(chan struct{})
	defer close(stopCh)
	go informerFactory.Start(stopCh)

	secretInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			fmt.Printf("add event received for %s/%s\n", obj.(*corev1.Secret).Namespace, obj.(*corev1.Secret).Name)
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			fmt.Printf("update event received, old : %+v, new %+v\n", old, new)
		},
		DeleteFunc: func(obj interface{}) {
			fmt.Printf("delete event received for %s/%s\n", obj.(*corev1.Secret).Namespace, obj.(*corev1.Secret).Name)
		},
	})

	fmt.Println("using an informer, listing all secrets from indexer")
	secrets, err := secretInformer.Lister().List(labels.Everything())
	if err != nil {
		panic(err)
	}

	for _, secret := range secrets {
		fmt.Printf("%s/%s secret\n", secret.Namespace, secret.Name)
	}

	<-stopCh // close the channel to stop the informer factory
}
