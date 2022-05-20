package pkg

import (
	"context"
	"fmt"
	apicorev1 "k8s.io/api/core/v1"
	"k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	icv1 "k8s.io/client-go/informers/core/v1"
	inv1 "k8s.io/client-go/informers/networking/v1beta1"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/listers/core/v1"
	networkingv1 "k8s.io/client-go/listers/networking/v1beta1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"reflect"
	"time"
)

const workNum = 5

type controller struct {
	client        kubernetes.Interface
	ingressLister networkingv1.IngressLister
	serviceLister corev1.ServiceLister
	queue         workqueue.RateLimitingInterface // 延迟队列
}

func (c *controller) addService(obj interface{}) {
	fmt.Println("add service")
	c.enqueue(obj)
}

func (c *controller) UpdateService(oldObj interface{}, newObj interface{}) {
	// todo 比较annotation是否相等
	if reflect.DeepEqual(oldObj, newObj) {
		return
	}
	fmt.Println("update service")
	c.enqueue(newObj)
}

func (c *controller) enqueue(obj interface{}) {
	// 获取唯一键放入队列等待处理
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(err)
	}
	c.queue.Add(key)
}

func (c *controller) deleteIngress(obj interface{}) {
	ingress := obj.(*v1beta1.Ingress)
	ownerReference := metav1.GetControllerOf(ingress)
	if ownerReference == nil {
		return
	}
	if ownerReference.Kind != "Service" {
		return
	}
	// 讲ingress转换成service处理 因为其他的都是入service类型的
	c.queue.Add(ingress.Namespace + "/" + ownerReference.Name)
}

func (c *controller) processNextItem() bool {
	item, shutdown := c.queue.Get()
	if shutdown {
		return false
	}
	defer c.queue.Done(item)

	key := item.(string)

	err := c.syncService(key)
	if err != nil {
		c.handlerError(key, err)
	}

	return true
}

func (c *controller) worker() {
	for c.processNextItem() {

	}
}

func (c *controller) syncService(key string) error {
	namespaceKey, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	// 如果错误是找不到的错误，就是service被删除了，都已经被删除了就不用处理了
	service, err := c.serviceLister.Services(namespaceKey).Get(name)
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}

	// 是否是需要处理的service
	_, ok := service.GetAnnotations()["ingress/http"]

	ingress, err := c.ingressLister.Ingresses(namespaceKey).Get(name)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	// 是需要处理的service，并且没有给他新增ingress，就处理新增一个ingress
	if ok && errors.IsNotFound(err) {
		fmt.Println("create ingress")
		ingress = c.constructIngress(service)
		_, err := c.client.NetworkingV1beta1().Ingresses(namespaceKey).Create(context.TODO(), ingress, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	} else if !ok && ingress != nil {
		fmt.Println("delete ingress")
		err := c.client.NetworkingV1beta1().Ingresses(namespaceKey).Delete(context.TODO(), name, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *controller) Run(stopCh <-chan struct{}) {
	for i := 0; i < workNum; i++ {
		go wait.Until(c.worker, time.Minute, stopCh)
	}
	<-stopCh
}

func (c *controller) handlerError(item string, err error) {
	// 重试了几次 大于这个次数就不重试了
	if c.queue.NumRequeues(item) <= 5 {
		// 错误了限制一下下次处理的时间
		c.queue.AddRateLimited(item)
		return
	}

	runtime.HandleError(err)
	c.queue.Forget(item)
}

func (c *controller) constructIngress(service *apicorev1.Service) *v1beta1.Ingress {
	ing := &v1beta1.Ingress{}

	// 加入附属关系
	ing.ObjectMeta.OwnerReferences = []metav1.OwnerReference{
		*metav1.NewControllerRef(service, apicorev1.SchemeGroupVersion.WithKind("Service")),
	}

	ing.Name = service.Name
	ing.Namespace = service.Namespace
	pathType := v1beta1.PathTypePrefix
	ing.Spec = v1beta1.IngressSpec{
		Rules: []v1beta1.IngressRule{
			{
				Host: "example.com",
				IngressRuleValue: v1beta1.IngressRuleValue{
					HTTP: &v1beta1.HTTPIngressRuleValue{
						Paths: []v1beta1.HTTPIngressPath{
							{
								Path:     "/",
								PathType: &pathType,
								Backend: v1beta1.IngressBackend{
									ServiceName: service.Name,
									ServicePort: intstr.FromInt(8080),
								},
							},
						},
					},
				}},
		},
	}
	return ing
}

func NewController(client kubernetes.Interface, ingressInformer inv1.IngressInformer, serviceInformer icv1.ServiceInformer) *controller {

	c := &controller{
		client:        client,
		ingressLister: ingressInformer.Lister(),
		serviceLister: serviceInformer.Lister(),
		queue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ingressManager"), // 把需要处理的数据放入队列
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
