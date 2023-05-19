package client

import (
	"fmt"
	"github.com/golang/glog"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/metrics/pkg/client/clientset/versioned"
	"sync"
)

type Client struct {
	Api        *kubernetes.Clientset
	MetricsApi *versioned.Clientset
}

var K8sClient *Client
var onceK8s sync.Once

//原生创建client
func NewClientK8s() {
	onceK8s.Do(func() {
		//本地配置信息
		cfg, err := clientcmd.BuildConfigFromFlags("", "/etc/config")
		if err != nil {
			glog.Errorf("kubernetes client failed")
			panic(err.Error())
		}
		K8sClient = &Client{Api: nil, MetricsApi: nil}
		//K8sClient.Api, err = kubernetes.NewForConfig(cfg)
		K8sClient.MetricsApi, err = versioned.NewForConfig(cfg)
		fmt.Println("k8s service success")
		if err != nil {
			glog.Errorf("kubernetes client failed")
			panic(err.Error())
		}
	})
}
