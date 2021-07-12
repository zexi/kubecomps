package types

import (
	"strings"

	"k8s.io/api/core/v1"
)

const (
	InstanceTypeVM      = "guest"
	InstanceTypeHost    = "host"
	InstanceTypeDefault = InstanceTypeHost
)

const (
	ServiceAnnotationLBCluster = "loadbalancer.yunion.io/cluster"
	ServiceAnnotationLBNetwork = "loadbalancer.yunion.io/network"
	ServiceAnnotationLBAddress = "loadbalancer.yunion.io/address"
)

const (
	LBStatusEnabled = "enabled"
)

const (
	LBBackendHost  = "host"
	LBBackendGuest = "guest"
)

const (
	NetworkTypeContainer = "container"
	NetworkTypeGuest     = "guest"
)

type SStandaloneResourceBase struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	ExternalId  string `json:"external_id"`
	Description string `json:"description"`
	IsEmulated  bool   `json:"is_emulated"`
}

func (b SStandaloneResourceBase) GetName() string {
	return b.Name
}

func (b SStandaloneResourceBase) GetId() string {
	return b.Id
}

type SStatusStandaloneResourceBase struct {
	SStandaloneResourceBase
	Status string `json:"status"`
}

type SVirtualResourceBase struct {
	SStatusStandaloneResourceBase
	ProjectId string `json:"tenant_id"`
	IsSystem  bool   `json:"is_system"`
}

type SSharableVirtualResourceBase struct {
	SVirtualResourceBase
	IsPublic bool `json:"is_public"`
}

type SHostNic struct {
	HostId  string `json:"baremetal_id"`
	Index   int    `json:"index"`
	Dns     string `json:"dns"`
	Domain  string `json:"domain"`
	Gateway string `json:"gateway"`
	IpAddr  string `json:"ip_addr"`
	LinkUp  bool   `json:"link_up"`
	Mac     string `json:"mac"`
	MaskLen int    `json:"masklen"`
	Net     string `json:"net"`
	NetId   string `json:"net_id"`
	NicType string `json:"nic_type"`
	Wire    string `json:"wire"`
	WireId  string `json:"wire_id"`
}

type SHost struct {
	SStatusStandaloneResourceBase
	Enabled     bool       `json:"enabled"`
	ManagerId   string     `json:"manager_id"`
	Rack        string     `json:"rack"`
	Slots       string     `json:"slots"`
	AccessMac   string     `json:"access_mac"`
	AccessIp    string     `json:"access_ip"`
	CpuCount    int        `json:"cpu_count"`
	MemSize     int        `json:"mem_size"`
	StorageSize int        `json:"storage_size"`
	HostStatus  string     `json:"host_status"`
	ZoneId      string     `json:"zone_id"`
	Zone        string     `json:"zone"`
	Nics        []SHostNic `json:"nic_info"`
}

type SHostStorage struct {
	StorageId   string `json:"storage_id"`
	HostId      string `json:"host_id"`
	Host        string `json:"host"`
	Name        string `json:"storage"`
	MediumType  string `json:"medium_type"`
	StorageType string `json:"storage_type"`
	//Status       string `json:"status"`
	Enabled      bool  `json:"enabled"`
	Capacity     int64 `json:"capacity"`
	FreeCapacity int64 `json:"free_capacity"`
}

type SNetwork struct {
	SSharableVirtualResourceBase
	Exit         bool   `json:"exit"`
	GuestDns     string `json:"guest_dns"`
	GuestIpStart string `json:"guest_ip_start"`
	GuestIpEnd   string `json:"guest_ip_end"`
	GuestIpMask  int    `json:"guest_ip_mask"`
	TotalCount   int    `json:"ports"`
	UsedCount    int    `json:"ports_used"`
	ServerType   string `json:"server_type"`
	Wire         string `json:"wire"`
	WireId       string `json:"wire_id"`
	VlanId       int    `json:"vlan_id"`
}

func (n SNetwork) GetFreeCount() int {
	return n.TotalCount - n.UsedCount
}

type SGuest struct {
	SVirtualResourceBase

	VcpuCount    int    `json:"vcpu_count"`
	VmemSize     int    `json:"vmem_size"`
	HostId       string `json:"host_id"`
	Hypervisor   string `json:"hypervisor"`
	InstanceType string `json:"instance_type"`
	Zone         string `json:"zone"`
	ZoneId       string `json:"zone_id"`
}

type SLoadBalancer struct {
	SVirtualResourceBase

	Address        string `json:"address"`
	AddressType    string `json:"address_type"`
	NetworkType    string `json:"network_type"`
	NetworkId      string `json:"network_id"`
	ZoneId         string `json:"zone_id"`
	BackendGroupId string `json:"backend_group_id"`
}

type SLBListener struct {
	SVirtualResourceBase
	LoadbalancerId string `json:"loadbalancer_id"`
	ListenerType   string `json:"listener_type"`
	ListenerPort   int    `json:"listener_port"`
	BackendGroupId string `json:"backend_group_id"`
}

func (l SLBListener) IsManagedPort(port v1.ServicePort) bool {
	return strings.ToLower(l.ListenerType) == strings.ToLower(string(port.Protocol)) && l.ListenerPort == int(port.Port)
}

func (l SLBListener) HasAnyPort(ports []v1.ServicePort) bool {
	for _, port := range ports {
		if l.IsManagedPort(port) {
			return true
		}
	}
	return false
}

type SLBListenerRule struct {
	SVirtualResourceBase

	ListenerId     string `json:"listener_Id"`
	BackendGroupId string `json:"backend_group_id"`
	Path           string `json:"path"`
	Domain         string `json:"domain"`
}

type SLBBackendGroup struct {
	SVirtualResourceBase

	LoadbalancerId string `json:"loadbalancer_id"`
}

type SLBBackend struct {
	SVirtualResourceBase
	BackendGroupId string `json:"backend_group_id"`
	BackendId      string `json:"backend_id"`
	BackendType    string `json:"backend_type"`
	Weight         int    `json:"weight"`
	Address        string `json:"address"`
	Port           int    `json:"port"`
}

type SWire struct {
	SStandaloneResourceBase
}

type SZone struct {
	SStandaloneResourceBase
	Cloudregion   string `json:"cloudregion"`
	CloudregionId string `json:"cloudregion_id"`
}

type SCloudRegion struct {
	SStandaloneResourceBase
}

type SDisk struct {
	SVirtualResourceBase
	AccessPath string `json:"access_path"`
	Format     string `json:"disk_format"`
	Size       int64  `json:"disk_size"`
	Type       string `json:"disk_type"`
	StorageId  string `json:"storage_id"`
	Storage    string `json:"storage"`
}

type CreateDiskInput struct {
	Name       string `json:"name"`
	StorageId  string `json:"storage_id"`
	Fs         string `json:"fs"`
	Size       int    `json:"size"`
	DiskType   string `json:"disk_type"`
	Backend    string `json:"backend"` // storageType
	Format     string `json:"format"`
	PreferHost string `json:"prefer_host"`
}
