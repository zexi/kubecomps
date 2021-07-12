package drivers

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"

	"yunion.io/x/kubecomps/pkg/cloudprovider/drivers"
	"yunion.io/x/kubecomps/pkg/types"
)

func init() {
	drivers.RegisterInstanceDriver(GuestDriver{})
}

// GuestDriver implements drivers.InstanceManager
type GuestDriver struct{}

func (m GuestDriver) GetDriverName() string {
	return types.InstanceTypeVM
}

func (m GuestDriver) GetByName(s *mcclient.ClientSession, name string) (drivers.Instance, error) {
	obj, err := modules.Servers.Get(s, name, nil)
	if err != nil {
		return nil, fmt.Errorf("Get vm by name %q: %v", name, err)
	}
	srv := new(Server)
	err = obj.Unmarshal(srv)
	return srv, err
}

// Server implemnets drviers.Instance
type Server struct {
	types.SGuest
}

func (srv Server) GetHost(s *mcclient.ClientSession) (*Host, error) {
	hostDrv := HostDriver{}
	host, err := hostDrv.GetByName(s, srv.HostId)
	if err != nil {
		return nil, fmt.Errorf("Get host %q error: %v", srv.HostId, err)
	}
	return host.(*Host), nil
}

func (srv Server) GetWire(s *mcclient.ClientSession) (drivers.Wire, error) {
	host, err := srv.GetHost(s)
	if err != nil {
		return nil, fmt.Errorf("Server %q get host error: %v", srv.GetName(), err)
	}
	return host.GetWire(s)
}

func (srv Server) GetZone(s *mcclient.ClientSession) (drivers.Zone, error) {
	host, err := srv.GetHost(s)
	if err != nil {
		return nil, fmt.Errorf("Server %q get host error: %v", srv.GetName(), err)
	}
	return host.GetZone(s)
}

func (srv Server) GetRegion(s *mcclient.ClientSession) (drivers.CloudRegion, error) {
	host, err := srv.GetHost(s)
	if err != nil {
		return nil, fmt.Errorf("Server %q get host error: %v", srv.GetName(), err)
	}
	return host.GetRegion(s)
}

func (srv Server) GetIPAddresses(s *mcclient.ClientSession) ([]string, error) {
	params := jsonutils.NewDict()
	params.Add(jsonutils.JSONTrue, "system")
	params.Add(jsonutils.JSONTrue, "admin")
	ret, err := modules.Servernetworks.ListDescendent(s, srv.GetId(), params)
	if err != nil {
		return nil, err
	}
	if len(ret.Data) == 0 {
		return nil, fmt.Errorf("Not found networks by server: %s", srv.GetName())
	}
	ips := []string{}
	for _, obj := range ret.Data {
		ip, _ := obj.GetString("ip_addr")
		if len(ip) != 0 {
			ips = append(ips, ip)
		}
	}
	return ips, nil
}

func (srv Server) GetCPUCount() int {
	return srv.VcpuCount
}

func (srv Server) GetMemorySize() int {
	return srv.VmemSize
}

func (srv Server) GetLoadBalancerBackendType() string {
	return types.LBBackendGuest
}
