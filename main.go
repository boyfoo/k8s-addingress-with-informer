package main

import (
	"fmt"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8splay/pkg"
	"log"
)

func main() {
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		clusterConfig, err := rest.InClusterConfig()
		if err != nil {
			log.Fatalln(err)
		}
		config = clusterConfig
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalln(err)
	}

	factory := informers.NewSharedInformerFactory(clientSet, 0)
	servicesInformer := factory.Core().V1().Services()
	ingressesInformer := factory.Networking().V1beta1().Ingresses()

	c := pkg.NewController(clientSet, ingressesInformer, servicesInformer)

	factory.Start(wait.NeverStop)
	factory.WaitForCacheSync(wait.NeverStop) // 等待同步

	c.Run(wait.NeverStop)

	fmt.Println("stop")
}
