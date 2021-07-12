package drivers

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"

	"yunion.io/x/kubecomps/pkg/cloudprovider/drivers"
	"yunion.io/x/kubecomps/pkg/types"
)

func init() {
	drivers.RegisterInstanceDriver(HostDriver{})
}

// HostDriver implements drivers.InstanceManager
type HostDriver struct{}

func (m HostDriver) GetDriverName() string {
	return types.InstanceTypeHost
}

func (m HostDriver) GetByName(s *mcclient.ClientSession, name string) (drivers.Instance, error) {
	obj, err := modules.Hosts.Get(s, name, nil)
	if err != nil {
		return nil, err
	}
	host := new(Host)
	err = obj.Unmarshal(host)
	return host, err
}

// Host implements drivers.Instance
type Host struct {
	types.SHost
}

func (h Host) GetWire(s *mcclient.ClientSession) (drivers.Wire, error) {
	if len(h.Nics) == 0 {
		return nil, fmt.Errorf("Host %s no nics", h.GetName())
	}
	ret, err := modules.Wires.Get(s, h.Nics[0].WireId, nil)
	if err != nil {
		return nil, err
	}
	wire := types.SWire{}
	err = ret.Unmarshal(&wire)
	return wire, err
}

func (h Host) GetZone(s *mcclient.ClientSession) (drivers.Zone, error) {
	return types.SZone{
		SStandaloneResourceBase: types.SStandaloneResourceBase{
			Id:   h.ZoneId,
			Name: h.Zone,
		},
	}, nil
}

func (h Host) GetRegion(s *mcclient.ClientSession) (drivers.CloudRegion, error) {
	ret, err := modules.Zones.Get(s, h.ZoneId, nil)
	if err != nil {
		return nil, err
	}
	zone := types.SZone{}
	if err := ret.Unmarshal(&zone); err != nil {
		return nil, err
	}
	regionObj, err := modules.Cloudregions.Get(s, zone.CloudregionId, nil)
	region := types.SCloudRegion{}
	if err := regionObj.Unmarshal(&regionObj); err != nil {
		return nil, err
	}
	return region, nil
}

func (h Host) GetIPAddresses(_ *mcclient.ClientSession) ([]string, error) {
	return []string{h.AccessIp}, nil
}

func (h Host) GetCPUCount() int {
	return h.CpuCount
}

func (h Host) GetMemorySize() int {
	return h.MemSize
}

func (h Host) GetLoadBalancerBackendType() string {
	return types.LBBackendHost
}
