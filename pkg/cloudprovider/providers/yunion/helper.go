package yunion

import (
	"fmt"

	"k8s.io/api/core/v1"

	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/kubecomps/pkg/cloudprovider/drivers"
	"yunion.io/x/kubecomps/pkg/types"
	"yunion.io/x/kubecomps/pkg/util/onecloud"
)

func IsNotFoundError(err error) bool {
	return onecloud.IsNotFoundError(err)
}

func GetInstanceAddressByName(man drivers.InstanceManager, s *mcclient.ClientSession, name string) ([]v1.NodeAddress, error) {
	ins, err := man.GetByName(s, name)
	if err != nil {
		return nil, err
	}
	ips, err := ins.GetIPAddresses(s)
	if err != nil {
		return nil, err
	}
	return toNodeAddresses(ips), nil
}

func toNodeAddresses(ips []string) []v1.NodeAddress {
	ret := make([]v1.NodeAddress, 0)
	for _, ip := range ips {
		ret = append(ret, v1.NodeAddress{
			Type:    v1.NodeInternalIP,
			Address: ip,
		})
	}
	return ret
}

func getListenerForPort(lis []*types.SLBListener, port v1.ServicePort) *types.SLBListener {
	for _, l := range lis {
		if l.IsManagedPort(port) {
			return l
		}
	}
	return nil
}

func getBackendGroupName(lbName string, port v1.ServicePort) string {
	return fmt.Sprintf("%s-%s-%d", lbName, port.Protocol, port.Port)
}

func createOrUpdateLBBackends(man drivers.InstanceManager, helper *onecloud.LBHelper, lb *types.SLoadBalancer, port v1.ServicePort, nodes []*v1.Node) (string, error) {
	groupName := getBackendGroupName(lb.Name, port)
	backends := make([]onecloud.LBBackend, 0)
	for _, n := range nodes {
		ins, err := man.GetByName(helper.Session(), n.Name)
		if err != nil {
			return "", err
		}
		backends = append(backends, onecloud.LBBackend{
			Id:   ins.GetId(),
			Type: ins.GetLoadBalancerBackendType(),
		})
	}
	group, err := helper.CreateOrUpdateLBBackends(groupName, lb, int(port.NodePort), backends)
	if err != nil {
		return "", fmt.Errorf("create or update backends: %v", err)
	}
	return group.GetId(), nil
}

func GetListenerName(lbName string, port v1.ServicePort) string {
	return fmt.Sprintf("%s-%s-%d", lbName, port.Protocol, port.Port)
}

func toServiceLBStatus(lb *types.SLoadBalancer) *v1.LoadBalancerStatus {
	return &v1.LoadBalancerStatus{
		[]v1.LoadBalancerIngress{
			v1.LoadBalancerIngress{
				IP: lb.Address,
			},
		},
	}
}

func GetNodeNames(nodes []*v1.Node) []string {
	hosts := []string{}
	for _, node := range nodes {
		hosts = append(hosts, node.Name)
	}
	return hosts
}

func findNetworkIDForLB(man drivers.InstanceManager, s *mcclient.ClientSession, node v1.Node) (string, error) {
	ins, err := man.GetByName(s, node.Name)
	if err != nil {
		return "", err
	}
	wire, err := ins.GetWire(s)
	if err != nil {
		return "", fmt.Errorf("Get instance %s wire error: %v", ins.GetName(), err)
	}
	zone, err := ins.GetZone(s)
	if err != nil {
		return "", fmt.Errorf("Get instance %s zone error: %v", ins.GetName(), err)
	}
	return onecloud.FindNetworkIDForLB(s, wire.GetId(), zone.GetId())
}
