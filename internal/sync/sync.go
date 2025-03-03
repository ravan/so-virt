package sync

import (
	"github.com/ravan/so-virt/internal/config"
	"github.com/ravan/so-virt/internal/virt"
	"github.com/ravan/stackstate-client/stackstate/receiver"
	"github.com/samber/lo"
	"k8s.io/apimachinery/pkg/types"
	v1 "kubevirt.io/api/core/v1"
)

const (
	Source = "virt"
	Pod    = "pod"
)

func Sync(conf *config.Configuration) (*receiver.Factory, error) {
	factory := receiver.NewFactory(Source, Source, conf.Kubernetes.Cluster)
	client, err := virt.New(conf)
	if err != nil {
		return nil, err
	}
	vmis, err := client.GetVirtualMachineInstances()
	if err != nil {
		return nil, err
	}
	lo.ForEach(vmis, func(vmi v1.VirtualMachineInstance, index int) {
		processVMI(&vmi, factory)
	},
	)
	return factory, nil
}

func processVMI(vmi *v1.VirtualMachineInstance, f *receiver.Factory) {
	c := mapVirtualMachine(vmi, f)
	mapVirtualMachineHost(vmi.Status.NodeName, c, f)
	lo.ForEach(lo.Keys(vmi.Status.ActivePods), func(p types.UID, index int) {
		mapVirtualMachinePod(string(p), c, f)
	})
}
