package sync

import (
	"fmt"
	"github.com/ravan/stackstate-client/stackstate/receiver"
	"github.com/samber/lo"
	v1 "kubevirt.io/api/core/v1"
	"log/slog"
	"sigs.k8s.io/yaml"
	"strings"
)

const (
	CTypeVM   = "vm"
	CTypePod  = "pod"
	CTypeNode = "node"
	Domain    = "Suse Virtualization"
)

func mapVirtualMachine(vmi *v1.VirtualMachineInstance, f *receiver.Factory) *receiver.Component {
	id := UrnVM(vmi.Name, vmi.Namespace, f.Cluster)
	var c *receiver.Component
	if f.ComponentExists(id) {
		c = f.MustGetComponent(id)
	} else {
		c = f.MustNewComponent(id, vmi.Name, CTypeVM)
		c.Data.Layer = "Virtual Machines"
		c.Data.Domain = f.Cluster
		c.AddProperty("vmPhase", vmi.Status.Phase)
		timestamp, found := lo.Find(vmi.Status.PhaseTransitionTimestamps, func(item v1.VirtualMachineInstancePhaseTransitionTimestamp) bool {
			return item.Phase == vmi.Status.Phase
		})
		if found {
			c.AddProperty("phaseTimestamp", timestamp.PhaseTransitionTimestamp)
		}
		c.AddProperty("guestInfo", vmi.Status.GuestOSInfo)
		c.AddProperty("cpuCores", vmi.Spec.Domain.CPU.Cores)
		c.AddProperty("memory", vmi.Spec.Domain.Memory.Guest)

		netinterface, found := lo.Find(vmi.Status.Interfaces, func(item v1.VirtualMachineInstanceNetworkInterface) bool {
			return item.Name == "default"
		})
		if found {
			c.AddProperty("vmIP", netinterface.IP)
		}
		c.AddProperty("guestInfo", vmi.Status.GuestOSInfo.PrettyName)
		addSourceProperties(vmi, c)
		addCommonLabels(vmi.Namespace, vmi.Status.NodeName, c, f)
	}
	return c
}

func mapVirtualMachinePod(podUid string, vmiComp *receiver.Component, f *receiver.Factory) *receiver.Component {
	id := UrnPod(podUid, f.Cluster)
	var c *receiver.Component
	if f.ComponentExists(id) {
		c = f.MustGetComponent(id)
	} else {
		c = f.MustNewComponent(id, podUid, CTypePod)
		c.Data.Layer = "Pods"
		c.Data.Domain = Domain
	}
	if !f.RelationExists(vmiComp.ID, id) {
		f.MustNewRelation(vmiComp.ID, id, "launched by")
	}
	return c
}

func mapVirtualMachineHost(nodeName string, vmiComp *receiver.Component, f *receiver.Factory) *receiver.Component {
	id := f.UrnNode(nodeName)
	var c *receiver.Component
	if f.ComponentExists(id) {
		c = f.MustGetComponent(id)
	} else {
		c = f.MustNewComponent(id, nodeName, CTypeNode)
		c.Data.Layer = "Nodes"
		c.Data.Domain = Domain
		c.AddProperty("vmCount", "0")
	}
	if !f.RelationExists(vmiComp.ID, id) {
		count := 0
		if _, ok := f.Lookup[id]; ok {
			count = f.Lookup[id].(int)
		}
		count += 1
		f.Lookup[id] = count
		c.AddProperty("vmCount", fmt.Sprintf("%d", count))
		f.MustNewRelation(vmiComp.ID, id, "runs on")
	}
	return c
}

func UrnVM(name, namespace, cluster string) string {
	urn := fmt.Sprintf("urn:susevirt:/%s:%s:vm/%s", cluster, namespace, name)
	return sanitizeUrn(urn)
}

func UrnPod(podUid, cluster string) string {
	urn := fmt.Sprintf("urn:kubernetes:/%s/pod/%s", cluster, podUid)
	return sanitizeUrn(urn)
}
func UrnCluster(cluster string) string {
	return fmt.Sprintf("urn:cluster:/kubernetes:%s", cluster)
}

func addCommonLabels(namespace, nodeName string, c *receiver.Component, f *receiver.Factory) {
	c.AddLabelKey("namespace", namespace)
	c.AddLabelKey("cluster-name", f.Cluster)
	c.AddLabelKey("node-name", nodeName)
	c.AddProperty("namespaceIdentifier", f.UrnNamespace(namespace))
	c.AddProperty("nodeIdentifier", f.UrnNode(nodeName))
	c.AddProperty("nodeName", nodeName)
	c.AddProperty("clusterIdentifier", UrnCluster(f.Cluster))

}

func addSourceProperties(obj any, c *receiver.Component) {
	bytes, err := yaml.Marshal(obj)
	result := make(map[string]interface{})
	if err != nil {
		slog.Error("failed to marshal object to yaml", slog.Any("error", obj))
		return
	}
	err = yaml.Unmarshal(bytes, &result)
	if err != nil {
		slog.Error("failed to unmarshal object to yaml", slog.Any("error", obj))
	}
	delete(result["metadata"].(map[string]any), "managedFields")
	c.SourceProperties = result

}

func sanitizeUrn(urn string) string {
	return strings.ToLower(strings.Replace(urn, " ", "_", -1))
}
