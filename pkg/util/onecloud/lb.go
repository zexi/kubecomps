package onecloud

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"yunion.io/x/jsonutils"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/wait"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"

	"yunion.io/x/kubecomps/pkg/types"
)

type LBHelper struct {
	*ResourceHelper
}

func NewLBHelper(s *mcclient.ClientSession) *LBHelper {
	return &LBHelper{
		ResourceHelper: NewResourceHelper(s, &modules.Loadbalancers),
	}
}

func (lb *LBHelper) GetByName(name string) (*types.SLoadBalancer, error) {
	opt := options.LoadbalancerListOptions{}
	opt.Filter = []string{fmt.Sprintf("name.in(%s)", name)}
	admin := true
	opt.Admin = &admin
	params, err := options.ListStructToParams(&opt)
	if err != nil {
		return nil, err
	}
	ret, err := modules.Loadbalancers.List(lb.session, params)
	if err != nil {
		return nil, err
	}
	if len(ret.Data) == 0 {
		// not found
		return nil, errors.New(NotFoundMsg)
	}
	if len(ret.Data) >= 2 {
		return nil, fmt.Errorf("Found %d duplicate name LBs", len(ret.Data))
	}
	lbObj := types.SLoadBalancer{}
	err = ret.Data[0].Unmarshal(&lbObj)
	return &lbObj, err
}

func (lb *LBHelper) EnsureDelete(id string) error {
	if _, err := lb.Delete(lb.session, id, nil); err != nil {
		return err
	}
	return lb.WaitObjectDelete(id, true, continueWaitResource)
}

func (lb *LBHelper) Create(name string, netId string, address string, cluster string) (*types.SLoadBalancer, error) {
	opt := options.LoadbalancerCreateOptions{
		NAME:    name,
		Network: netId,
		Address: address,
		Cluster: cluster,
	}
	params, err := options.StructToParams(&opt)
	if err != nil {
		return nil, err
	}
	ret, err := lb.Manager.Create(lb.session, params)
	if err != nil {
		return nil, fmt.Errorf("Create lb: %v", err)
	}
	lbObj := new(types.SLoadBalancer)
	if err := ret.Unmarshal(lbObj); err != nil {
		return nil, fmt.Errorf("Unmarshal lb: %v", err)
	}
	return lb.waitEnabled(lbObj.GetId())
}

func (lb *LBHelper) waitEnabled(id string) (*types.SLoadBalancer, error) {
	interval := 5 * time.Second
	timeout := 5 * time.Minute
	lbObj := new(types.SLoadBalancer)
	err := wait.Poll(interval, timeout, func() (bool, error) {
		ret, err := modules.Loadbalancers.Get(lb.session, id, nil)
		if err != nil {
			log.Errorf("Get Loadbalancers %s: %v", id, err)
			return false, err
		}
		if err := ret.Unmarshal(lbObj); err != nil {
			log.Errorf("Unmarshal Loadbalancers %s: %v", id, err)
			return false, err
		}
		if strings.HasSuffix(lbObj.Status, "failed") {
			return false, fmt.Errorf("LB %s bad status %s", lbObj.Name, lbObj.Status)
		}
		if lbObj.Status != types.LBStatusEnabled {
			log.Infof("LB %s status %s != %s, continue wait...", lbObj.Name, lbObj.Status, types.LBStatusEnabled)
			return false, nil
		}
		return true, nil
	})
	return lbObj, err
}

func (lb *LBHelper) GetListenersByLoadBalancerID(lbId string) ([]*types.SLBListener, error) {
	opt := options.LoadbalancerListenerListOptions{
		Loadbalancer: lbId,
	}
	isAdmin := true
	opt.Admin = &isAdmin
	params, err := options.ListStructToParams(&opt)
	if err != nil {
		return nil, err
	}
	ret, err := modules.LoadbalancerListeners.List(lb.session, params)
	if err != nil {
		return nil, err
	}
	listeners := []*types.SLBListener{}
	for _, obj := range ret.Data {
		lis := types.SLBListener{}
		err = obj.Unmarshal(&lis)
		if err != nil {
			log.Errorf("Unmarshal LB listeners error: %v", err)
			continue
		}
		listeners = append(listeners, &lis)
	}
	return listeners, nil
}

func (lb *LBHelper) GetBackendsByGroupId(groupId string) ([]*types.SLBBackend, error) {
	opt := options.LoadbalancerBackendListOptions{
		BackendGroup: groupId,
	}
	isAdmin := true
	opt.Admin = &isAdmin
	params, err := options.ListStructToParams(&opt)
	if err != nil {
		return nil, err
	}
	ret, err := modules.LoadbalancerBackends.List(lb.session, params)
	if err != nil {
		return nil, err
	}
	backends := []*types.SLBBackend{}
	for _, obj := range ret.Data {
		b := types.SLBBackend{}
		err = obj.Unmarshal(&b)
		if err != nil {
			log.Errorf("Unmarshal LB backend error: %v", err)
			continue
		}
		backends = append(backends, &b)
	}
	return backends, nil
}

func (lb *LBHelper) GetLBBackendGroupByName(name string) (*types.SLBBackendGroup, error) {
	ret, err := modules.LoadbalancerBackendGroups.Get(lb.session, name, nil)
	if err != nil {
		return nil, err
	}
	obj := types.SLBBackendGroup{}
	err = ret.Unmarshal(&obj)
	return &obj, err
}

func (lb *LBHelper) GetLBBackend(groupId string, port int, backendType string, backendId string) (*types.SLBBackend, error) {
	opt := options.LoadbalancerBackendListOptions{
		BackendGroup: groupId,
		Port:         &port,
	}
	opt.Filter = []string{
		fmt.Sprintf("backend_type.equals(%s)", backendType),
		fmt.Sprintf("backend_id.equals(%s)", backendId),
	}
	isAdmin := true
	opt.Admin = &isAdmin
	params, err := options.ListStructToParams(&opt)
	ret, err := modules.LoadbalancerBackends.List(lb.session, params)
	if err != nil {
		return nil, err
	}
	if len(ret.Data) == 0 {
		return nil, errors.New(NotFoundMsg)
	}
	b := types.SLBBackend{}
	err = ret.Data[0].Unmarshal(&b)
	return &b, err
}

func (lb *LBHelper) CreateLBBackendGroup(lbObj *types.SLoadBalancer, groupName string) (*types.SLBBackendGroup, error) {
	opt := &options.LoadbalancerBackendGroupCreateOptions{
		NAME:         groupName,
		Loadbalancer: lbObj.GetId(),
	}
	params, err := opt.Params()
	if err != nil {
		return nil, err
	}
	ret, err := modules.LoadbalancerBackendGroups.Create(lb.session, params)
	if err != nil {
		return nil, err
	}
	obj := types.SLBBackendGroup{}
	err = ret.Unmarshal(&obj)
	return &obj, err
}

func (lb *LBHelper) DeleteBackends(backends []*types.SLBBackend) error {
	for _, backend := range backends {
		if _, err := modules.LoadbalancerBackends.Delete(lb.session, backend.GetId(), nil); err != nil {
			return fmt.Errorf("Delete backend %s: %v", backend.GetId(), err)
		}
		if err := WaitObjectDelete(lb.session, &modules.LoadbalancerBackends, backend.GetId(), true, continueWaitResource); err != nil {
			return fmt.Errorf("Wait backend %s deleted: %v", backend.GetName(), err)
		}
	}
	return nil
}

func (lb *LBHelper) DeleteBackendGroup(id string) error {
	backends, err := lb.GetBackendsByGroupId(id)
	if err != nil {
		return fmt.Errorf("Found backend group %q backends: %v", id, err)
	}
	if err := lb.DeleteBackends(backends); err != nil {
		return err
	}
	_, err = modules.LoadbalancerBackendGroups.Delete(lb.session, id, nil)
	return err
}

func (lb *LBHelper) DeleteListenerBackends(l *types.SLBListener) error {
	groupId := l.BackendGroupId
	if _, err := modules.LoadbalancerListeners.Delete(lb.session, l.Id, nil); err != nil {
		return fmt.Errorf("Delete listener %s: %v", l.GetName(), err)
	}
	if err := WaitObjectDelete(lb.session, &modules.LoadbalancerListeners, l.Id, true, continueWaitResource); err != nil {
		return fmt.Errorf("Wait listener %s delete: %v", l.GetName(), err)
	}
	if groupId != "" {
		err := lb.DeleteBackendGroup(groupId)
		if err != nil {
			return err
		}
	}
	return nil
}

func (lb *LBHelper) CreateLBBackend(
	groupId string,
	port int,
	backendType string,
	backendId string,
) (*types.SLBBackend, error) {
	opt := options.LoadbalancerBackendCreateOptions{
		BackendGroup: groupId,
		BackendType:  backendType,
		Backend:      backendId,
		Port:         &port,
	}
	params, err := options.StructToParams(&opt)
	if err != nil {
		return nil, err
	}
	ret, err := modules.LoadbalancerBackends.Create(lb.session, params)
	if err != nil {
		return nil, err
	}
	obj := types.SLBBackend{}
	err = ret.Unmarshal(&obj)
	return &obj, err
}

func (lb *LBHelper) UpdateLBBackend(id string, portNum int) (*types.SLBBackend, error) {
	opt := options.LoadbalancerBackendUpdateOptions{
		Port: &portNum,
	}
	params, err := options.StructToParams(&opt)
	if err != nil {
		return nil, err
	}
	ret, err := modules.LoadbalancerBackends.Update(lb.session, id, params)
	if err != nil {
		return nil, err
	}
	obj := types.SLBBackend{}
	err = ret.Unmarshal(&obj)
	return &obj, err
}

func (lb *LBHelper) CreateListener(
	name string,
	lbId string,
	protocol string,
	port int,
	backendGroupId string,
) (*types.SLBListener, error) {
	protocol = strings.ToLower(string(protocol))
	portNum := port
	opt := options.LoadbalancerListenerCreateOptions{
		NAME:         name,
		Loadbalancer: lbId,
		ListenerType: protocol,
		ListenerPort: &portNum,
		BackendGroup: backendGroupId,
		Scheduler:    "wrr",
	}
	params, err := options.StructToParams(&opt)
	if err != nil {
		return nil, err
	}
	ret, err := modules.LoadbalancerListeners.Create(lb.session, params)
	if err != nil {
		return nil, err
	}
	obj := types.SLBListener{}
	err = ret.Unmarshal(&obj)
	return &obj, err
}

func (lb *LBHelper) UpdateListener(id string, name string, backendGroupId string) (*types.SLBListener, error) {
	opt := options.LoadbalancerListenerUpdateOptions{
		Name:         name,
		BackendGroup: backendGroupId,
		Scheduler:    "wrr",
	}
	params, err := options.StructToParams(&opt)
	if err != nil {
		return nil, err
	}
	ret, err := modules.LoadbalancerListeners.Update(lb.session, id, params)
	if err != nil {
		return nil, err
	}
	obj := types.SLBListener{}
	err = ret.Unmarshal(&obj)
	return &obj, err
}

func (lb *LBHelper) CreateListenerRule(listener *types.SLBListener, bgId, domain string, path string) (*types.SLBListenerRule, error) {
	name := fmt.Sprintf("%s-%s-%s", listener.GetName(), domain, path)
	opt := options.LoadbalancerListenerRuleCreateOptions{
		NAME:         name,
		Listener:     listener.GetId(),
		Domain:       domain,
		Path:         path,
		BackendGroup: bgId,
	}
	params, err := options.StructToParams(&opt)
	if err != nil {
		return nil, err
	}
	ret, err := modules.LoadbalancerListenerRules.Create(lb.session, params)
	if err != nil {
		return nil, err
	}
	obj := types.SLBListenerRule{}
	err = ret.Unmarshal(&obj)
	return &obj, err
}

type LBBackend struct {
	Id   string
	Type string
}

func (lb *LBHelper) CreateOrUpdateLBBackends(
	name string,
	lbObj *types.SLoadBalancer,
	targetPort int,
	backends []LBBackend,
) (*types.SLBBackendGroup, error) {
	group, err := lb.GetLBBackendGroupByName(name)
	if err != nil {
		if !IsNotFoundError(err) {
			return nil, fmt.Errorf("Get lb backend group: %v", err)
		}
		group, err = lb.CreateLBBackendGroup(lbObj, name)
		if err != nil {
			return nil, fmt.Errorf("Create lb backend group %s: %v", name, err)
		}
	}
	for _, backend := range backends {
		oldBackend, err := lb.GetLBBackend(group.GetId(), targetPort, backend.Type, backend.Id)
		if err != nil {
			if !IsNotFoundError(err) {
				return nil, fmt.Errorf("get lb backend: %v", err)
			}
		}
		if oldBackend == nil {
			_, err := lb.CreateLBBackend(group.GetId(), targetPort, backend.Type, backend.Id)
			if err != nil {
				return nil, fmt.Errorf("create lb backend: %v", err)
			}
		} else {
			_, err := lb.UpdateLBBackend(oldBackend.GetId(), targetPort)
			if err != nil {
				return nil, fmt.Errorf("update lb backend: %v", err)
			}
		}
	}
	return group, nil
}

func (lb *LBHelper) UpdateLoadBalancerListenerRuleBackends(lbId string, backends []LBBackend) error {
	listeners, err := lb.GetListenersByLoadBalancerID(lbId)
	if err != nil {
		return fmt.Errorf("Get listeners: %v", err)
	}
	for _, listener := range listeners {
		if err := lb.UpdateListenerRuleBackends(listener, backends); err != nil {
			return fmt.Errorf("Update listener %s backends: %v", listener.GetName(), err)
		}
	}
	return nil
}

func (lb *LBHelper) GetListenerRulesByListener(listener *types.SLBListener) ([]*types.SLBListenerRule, error) {
	opt := options.LoadbalancerListenerRuleListOptions{
		Listener: listener.GetId(),
	}
	isAdmin := true
	opt.Admin = &isAdmin
	params, err := options.ListStructToParams(&opt)
	if err != nil {
		return nil, err
	}
	ret, err := modules.LoadbalancerListenerRules.List(lb.session, params)
	if err != nil {
		return nil, err
	}
	rules := []*types.SLBListenerRule{}
	for _, obj := range ret.Data {
		rule := types.SLBListenerRule{}
		err = obj.Unmarshal(&rule)
		if err != nil {
			log.Errorf("Unmarshal LB listeners error: %v", err)
			continue
		}
		rules = append(rules, &rule)
	}
	return rules, nil
}

func (lb *LBHelper) UpdateListenerRuleBackends(listener *types.SLBListener, backends []LBBackend) error {
	rules, err := lb.GetListenerRulesByListener(listener)
	if err != nil {
		return fmt.Errorf("Get rules by listener %s: %v", listener.GetId(), err)
	}
	for _, rule := range rules {
		groupId := rule.BackendGroupId
		if err := lb.UpdateBackendGroupBackends(groupId, backends); err != nil {
			return fmt.Errorf("Update backendgroup backends: %v", err)
		}
	}

	return nil
}

func (lb *LBHelper) UpdateBackendGroupBackends(groupId string, backends []LBBackend) error {
	oldBackends, err := lb.GetBackendsByGroupId(groupId)
	if err != nil {
		return fmt.Errorf("Get backends by groupId %s: %v", groupId, err)
	}
	if len(oldBackends) == 0 {
		return nil
	}
	targetPort := oldBackends[0].Port
	// TODO: impl rolling update
	if err := lb.DeleteBackends(oldBackends); err != nil {
		return err
	}
	for _, be := range backends {
		if _, err := lb.CreateLBBackend(groupId, targetPort, be.Type, be.Id); err != nil {
			return fmt.Errorf("Create backend %#v %d for group %s: %v", be, targetPort, groupId, err)
		}
	}
	return nil
}

func FindNetworkIDForLB(s *mcclient.ClientSession, wireId string, zoneId string) (string, error) {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(fmt.Sprintf("server_type.in(%s,%s)", types.NetworkTypeContainer, types.NetworkTypeGuest)), "filter.0")
	params.Add(jsonutils.NewString(zoneId), "zone")
	ret, err := modules.Networks.ListInContext(s, params, &modules.Wires, wireId)
	if err != nil {
		return "", err
	}
	if len(ret.Data) == 0 {
		return "", fmt.Errorf("Can't find networks by host wire %q", wireId)
	}
	nets := []types.SNetwork{}
	for _, netObj := range ret.Data {
		net := types.SNetwork{}
		err = netObj.Unmarshal(&net)
		if err != nil {
			return "", err
		}
		if net.GetFreeCount() <= 0 {
			log.Warningf("network %q no free ports, skip it", net.Name)
			continue
		}
		nets = append(nets, net)
	}

	if len(nets) == 0 {
		return "", fmt.Errorf("Not found available network")
	}
	selectNet := nets[0]
	log.Infof("Use network %#v", selectNet)

	return selectNet.Id, nil
}
