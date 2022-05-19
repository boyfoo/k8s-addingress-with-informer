package pkg

import (
	"k8s.io/api/networking/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	icv1 "k8s.io/client-go/informers/core/v1"
	inv1 "k8s.io/client-go/informers/networking/v1beta1"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/listers/core/v1"
	networkingv1 "k8s.io/client-go/listers/networking/v1beta1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"reflect"
)

type controller struct {
	client        kubernetes.Interface
	ingressLister networkingv1.IngressLister
	serviceLister corev1.ServiceLister
	queue         workqueue.RateLimitingInterface // 延迟队列
}

func (c *controller) addService(obj interface{}) {
	c.enqueue(obj)
}

func (c *controller) UpdateService(oldObj interface{}, newObj interface{}) {
	// todo 比较annotation是否相等
	if reflect.DeepEqual(oldObj, newObj) {
		return
	}
	c.enqueue(newObj)
}

func (c *controller) enqueue(obj interface{}) {
	// 除去唯一键放入队列等待处理
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(err)
	}
	c.queue.Add(key)
}

func (c *controller) deleteIngress(obj interface{}) {
	ingress := obj.(*v1beta1.Ingress)
	service := v1.GetControllerOf(ingress)
	if service == nil {
		return
	}
	if service.Kind != "Service" {
		return
	}
	// 讲ingress转换成service处理 因为其他的都是入service类型的
	c.queue.Add(ingress.Namespace + "/" + service.Name)
}

func (c *controller) Run(stopCh <-chan struct{}) {
	<-stopCh
}

func NewController(client kubernetes.Interface, ingressInformer inv1.IngressInformer, serviceInformer icv1.ServiceInformer) *controller {

	c := &controller{
		client:        client,
		ingressLister: ingressInformer.Lister(),
		serviceLister: serviceInformer.Lister(),
		queue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ingressManager"),
	}

	serviceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addService,
		UpdateFunc: c.UpdateService,
	})

	ingressInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: c.deleteIngress,
	})

	return c
}
