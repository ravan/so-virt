package virt

import (
	"context"
	"github.com/ravan/so-virt/internal/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	v1 "kubevirt.io/api/core/v1"
	kubevirtclient "kubevirt.io/client-go/kubecli"
	"log"
)

type Virt struct {
	client kubevirtclient.KubevirtClient
}

func New(conf *config.Configuration) (*Virt, error) {
	var c *rest.Config
	var err error
	if conf.Kubernetes.InCluster {
		c, err = rest.InClusterConfig()
	} else {
		c, err = clientcmd.BuildConfigFromFlags("", conf.Kubernetes.KubeConfig)
	}
	if err != nil {
		return nil, err
	}
	intercept(c)
	client, err := kubevirtclient.GetKubevirtClientFromRESTConfig(c)
	if err != nil {
		log.Fatalf("cannot obtain KubeVirt client: %v\n", err)
	}

	return &Virt{
		client,
	}, nil
}

func (v *Virt) GetVirtualMachineInstances() ([]v1.VirtualMachineInstance, error) {
	// Retrieve all VirtualMachines from all namespaces
	vms, err := v.client.VirtualMachineInstance(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return vms.Items, nil
}
