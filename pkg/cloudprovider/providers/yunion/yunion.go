package yunion

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	k8stypes "k8s.io/apimachinery/pkg/types"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/klog"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"

	"yunion.io/x/kubecomps/pkg/cloudprovider/drivers"
	"yunion.io/x/kubecomps/pkg/types"

	_ "yunion.io/x/kubecomps/pkg/cloudprovider/drivers/instance"
)

const (
	ProviderName = "yunion"
)

func init() {
	cloudprovider.RegisterCloudProvider(ProviderName, func(config io.Reader) (cloudprovider.Interface, error) {
		cfg, err := ReadConfig(config)
		if err != nil {
			return nil, err
		}
		return NewYunionProvider(cfg), nil
	})
}

func ReadConfig(config io.Reader) (types.Config, error) {
	conf := types.Config{}
	err := json.NewDecoder(config).Decode(&conf)
	return conf, err
}

type YunionProvider struct {
	config types.Config
}

func NewYunionProvider(cfg types.Config) *YunionProvider {
	authInfo := auth.NewAuthInfo(cfg.AuthURL, "", cfg.AdminUser, cfg.AdminPassword, cfg.AdminProject, "")
	auth.Init(authInfo, false, true, "", "")
	if len(cfg.InstanceType) == 0 {
		cfg.InstanceType = types.InstanceTypeDefault
	}
	return &YunionProvider{
		config: cfg,
	}
}

func (p *YunionProvider) GetClient() *mcclient.ClientSession {
	return auth.GetAdminSessionWithPublic(context.TODO(), p.config.Region, "")
}

// nitialize passes a Kubernetes clientBuilder interface to the cloud provider
func (p *YunionProvider) Initialize(clientBuilder cloudprovider.ControllerClientBuilder, stop <-chan struct{}) {

}

// Clusters is a no-op
func (p *YunionProvider) Clusters() (cloudprovider.Clusters, bool) {
	return nil, false
}

// ProviderName returns the cloud provider ID
func (p *YunionProvider) ProviderName() string {
	return ProviderName
}

// ScrubDNS filters DNS settings for pods
func (p *YunionProvider) ScrubDNS(nameServers, searches []string) ([]string, []string) {
	return nameServers, searches
}

// HasClusterID returns true if the cluster has a clusterID
func (p *YunionProvider) HasClusterID() bool {
	return true
}

// Zones indicates that we support zones
func (p *YunionProvider) Zones() (cloudprovider.Zones, bool) {
	return p, true
}

func (p *YunionProvider) GetZone(ctx context.Context) (cloudprovider.Zone, error) {
	return cloudprovider.Zone{}, fmt.Errorf("GetZone not impl")
}

func (p *YunionProvider) GetZoneByProviderID(ctx context.Context, providerID string) (cloudprovider.Zone, error) {
	return cloudprovider.Zone{}, fmt.Errorf("GetZoneByProviderID not impl")
}

func (p *YunionProvider) InstanceManager() drivers.InstanceManager {
	man, err := drivers.GetInstanceManager(p.config.InstanceType)
	if err != nil {
		panic(fmt.Sprintf("Get instance driver %s", p.config.InstanceType))
	}
	return man
}

func (p *YunionProvider) GetZoneByNodeName(ctx context.Context, nodeName k8stypes.NodeName) (cloudprovider.Zone, error) {
	cli := p.GetClient()
	ins, err := p.InstanceManager().GetByName(cli, string(nodeName))
	if err != nil {
		return cloudprovider.Zone{}, err
	}
	zoneObj, err := ins.GetZone(cli)
	if err != nil {
		return cloudprovider.Zone{}, fmt.Errorf("Get instance zone: %v", err)
	}
	regionObj, err := ins.GetRegion(cli)
	if err != nil {
		return cloudprovider.Zone{}, fmt.Errorf("Get instance region: %v", err)
	}
	zone := cloudprovider.Zone{
		FailureDomain: zoneObj.GetName(),
		//Region:        p.config.Region,
		Region: regionObj.GetName(),
	}
	klog.Infof("The instance %s in zone %#v", ins.GetName(), zone)
	return zone, nil
}

func (p *YunionProvider) Routes() (cloudprovider.Routes, bool) {
	return nil, false
}

type ResourceManager struct {
	Provider *YunionProvider
}

func NewResourceManager(p *YunionProvider) *ResourceManager {
	return &ResourceManager{
		Provider: p,
	}
}

func (man *ResourceManager) GetClient() *mcclient.ClientSession {
	return man.Provider.GetClient()
}

func (man *ResourceManager) InstanceManager() drivers.InstanceManager {
	return man.Provider.InstanceManager()
}
