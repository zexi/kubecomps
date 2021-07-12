package drivers

import (
	"yunion.io/x/onecloud/pkg/mcclient"
)

type resourceManager interface {
	GetDriverName() string
}

type InstanceManager interface {
	resourceManager

	GetByName(s *mcclient.ClientSession, name string) (Instance, error)
}

type IResource interface {
	GetId() string
	GetName() string
}

type Instance interface {
	IResource

	GetWire(s *mcclient.ClientSession) (Wire, error)
	GetZone(s *mcclient.ClientSession) (Zone, error)
	GetRegion(s *mcclient.ClientSession) (CloudRegion, error)
	GetIPAddresses(s *mcclient.ClientSession) ([]string, error)
	GetCPUCount() int
	GetMemorySize() int

	GetLoadBalancerBackendType() string
}

type Wire interface {
	IResource
}

type Zone interface {
	IResource
}

type CloudRegion interface {
	IResource
}
