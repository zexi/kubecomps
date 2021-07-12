package yunion

import (
	"context"
	"fmt"

	"k8s.io/api/core/v1"
	cloudprovider "k8s.io/cloud-provider"

	"yunion.io/x/log"

	"yunion.io/x/kubecomps/pkg/cloudprovider/drivers"
	"yunion.io/x/kubecomps/pkg/types"
	"yunion.io/x/kubecomps/pkg/util/onecloud"
)

// LoadBalancer initializes a LoadBalancer object
func (p *YunionProvider) LoadBalancer() (cloudprovider.LoadBalancer, bool) {
	return NewLoadBalancerManager(p), true
}

type LoadBalancerManager struct {
	*ResourceManager
}

func NewLoadBalancerManager(p *YunionProvider) *LoadBalancerManager {
	return &LoadBalancerManager{
		ResourceManager: NewResourceManager(p),
	}
}

func (lb *LoadBalancerManager) GetLoadBalancer(ctx context.Context, clusterName string, service *v1.Service) (*v1.LoadBalancerStatus, bool, error) {
	lbName := lb.GetLoadBalancerName(ctx, clusterName, service)
	helper := onecloud.NewLBHelper(lb.GetClient())
	lbObj, err := helper.GetByName(lbName)
	if err != nil {
		if IsNotFoundError(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return toServiceLBStatus(lbObj), true, nil
}

// EnsureLoadBalancer creates a new load balancer 'name', or updates the existing one.
func (lb *LoadBalancerManager) EnsureLoadBalancer(ctx context.Context, clusterName string, apiService *v1.Service, nodes []*v1.Node) (*v1.LoadBalancerStatus, error) {
	if len(nodes) == 0 {
		return nil, fmt.Errorf("there are no available nodes for LoadBalancer service %s/%s", apiService.Namespace, apiService.Name)
	}
	hosts := GetNodeNames(nodes)

	log.Infof("EnsureLoadBalancer(%v, %v, %v, %v, %v, %v, %s)", clusterName, apiService.Namespace, apiService.Name, apiService.Spec.LoadBalancerIP, apiService.Spec.Ports, apiService.Annotations, hosts)

	ports := apiService.Spec.Ports
	if len(ports) == 0 {
		return nil, fmt.Errorf("no ports provided to load balancer")
	}
	var err error
	netID := getStringFromAnnotation(apiService.Annotations, types.ServiceAnnotationLBNetwork, "")
	if len(netID) == 0 {
		netID, err = findNetworkIDForLB(lb.InstanceManager(), lb.Provider.GetClient(), *nodes[0])
		if err != nil {
			return nil, fmt.Errorf("Failed to find network for service %s, node %s", apiService.Name, nodes[0].Name)
		}
	}
	address := getStringFromAnnotation(apiService.Annotations, types.ServiceAnnotationLBAddress, "")
	lbCluster := getStringFromAnnotation(apiService.Annotations, types.ServiceAnnotationLBCluster, "")

	lbName := lb.GetLoadBalancerName(ctx, clusterName, apiService)
	helper := onecloud.NewLBHelper(lb.GetClient())
	lbObj, err := helper.GetByName(lbName)
	if err != nil {
		if !IsNotFoundError(err) {
			return nil, fmt.Errorf("Failed to getting loadbalancer %s: %v", lbName, err)
		}
		log.V(2).Infof("Creating loadbalancer %q", lbName)
		lbObj, err = helper.Create(lbName, netID, address, lbCluster)
		if err != nil {
			// Unknown error, retry later
			return nil, fmt.Errorf("error creating loadbalancer %s: %v", lbName, err)
		}
	} else {
		log.V(2).Infof("LoadBalancerManager %q already exists", lbName)
	}

	err = syncLoadBalancer(lb.InstanceManager(), helper, lbObj, apiService, nodes)
	if err != nil {
		return nil, err
	}

	log.Infof("Sync loadbalancer: %#v", lbObj)
	return toServiceLBStatus(lbObj), nil
}

func (lb *LoadBalancerManager) GetLoadBalancerName(ctx context.Context, clusterName string, service *v1.Service) string {
	defaultName := cloudprovider.DefaultLoadBalancerName(service)
	return fmt.Sprintf("%s-%s-%s", service.Namespace, service.Name, defaultName[:8])
}

func getStringFromAnnotation(annon map[string]string, key string, defaultVal string) string {
	if val, ok := annon[key]; ok {
		return val
	}
	return defaultVal
}

func (lb *LoadBalancerManager) EnsureLoadBalancerDeleted(ctx context.Context, clusterName string, service *v1.Service) error {
	lbName := lb.GetLoadBalancerName(ctx, clusterName, service)
	log.Infof("EnsureLoadBalancerDeleted(%v, %v)", service.Name, lbName)
	helper := onecloud.NewLBHelper(lb.GetClient())
	lbObj, err := helper.GetByName(lbName)
	if err != nil && !IsNotFoundError(err) {
		return err
	}
	if lbObj == nil {
		return nil
	}
	return helper.EnsureDelete(lbObj.Id)
}

func (lb *LoadBalancerManager) UpdateLoadBalancer(ctx context.Context, clusterName string, service *v1.Service, nodes []*v1.Node) error {
	lbName := lb.GetLoadBalancerName(ctx, clusterName, service)
	log.Infof("UpdateLoadBalancer(%v, %v)", service.Name, lbName)

	helper := onecloud.NewLBHelper(lb.GetClient())
	lbObj, err := helper.GetByName(lbName)
	if err != nil {
		return err
	}
	if lbObj == nil {
		return fmt.Errorf("loadbalancer %s not exist", lbName)
	}

	return syncLoadBalancer(lb.InstanceManager(), helper, lbObj, service, nodes)
}

func syncLoadBalancer(man drivers.InstanceManager, helper *onecloud.LBHelper, lbObj *types.SLoadBalancer, service *v1.Service, nodes []*v1.Node) error {
	oldListeners, err := helper.GetListenersByLoadBalancerID(lbObj.Id)
	if err != nil {
		return fmt.Errorf("Fail get LB %q listeners: %v", lbObj.Name, err)
	}
	ports := service.Spec.Ports

	// create or update backends and listener
	for _, port := range ports {
		backendGroupId, err := createOrUpdateLBBackends(man, helper, lbObj, port, nodes)
		if err != nil {
			return fmt.Errorf("Failed to create or update backends on lb %q for port %#v: %v", lbObj.Name, port, err)
		}

		listenerName := GetListenerName(lbObj.Name, port)
		listener := getListenerForPort(oldListeners, port)
		if listener == nil {
			log.V(4).Infof("Creating listener for port %#v", port)
			_, err = helper.CreateListener(listenerName, lbObj.Id, string(port.Protocol), int(port.Port), backendGroupId)
			if err != nil {
				return fmt.Errorf("Failed to create listener %q on lb %s for port %#v", listenerName, lbObj.Name, port)
			}
		} else {
			_, err = helper.UpdateListener(listener.Id, listenerName, backendGroupId)
			if err != nil {
				return fmt.Errorf("Failed to update listener %q on lb %s for port %#v", listenerName, lbObj.Name, port)
			}
		}
	}

	// delete unused listener and backends
	for _, oldListener := range oldListeners {
		if !oldListener.HasAnyPort(ports) {
			err = helper.DeleteListenerBackends(oldListener)
			if err != nil {
				log.Errorf("Delete listener %s error: %v", oldListener.Name, err)
			}
		}
	}
	return err
}
