package yunion

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	cloudprovider "k8s.io/cloud-provider"

	"yunion.io/x/log"

	"yunion.io/x/kubecomps/pkg/cloudprovider/drivers"
)

type InstanceManager struct {
	*ResourceManager
}

func NewInstanceManager(p *YunionProvider) *InstanceManager {
	return &InstanceManager{
		ResourceManager: NewResourceManager(p),
	}
}

func (p *YunionProvider) Instances() (cloudprovider.Instances, bool) {
	return NewInstanceManager(p), true
}

// CurrentNodeName implements Instances.CurrentNodeName
// Note this is *not* necessarily the same as hostname
func (i *InstanceManager) CurrentNodeName(ctx context.Context, hostname string) (types.NodeName, error) {
	return types.NodeName(hostname), nil
}

// AddSSHKeyToAllInstances is not implemented
func (i *InstanceManager) AddSSHKeyToAllInstances(ctx context.Context, user string, keyData []byte) error {
	return cloudprovider.NotImplemented
}

// InstanceExistsByProviderID returns true if the instance with the given provider id still exist.
// If false is returned with no error, the instance will be immediately deleted by the cloud controller manager
func (i *InstanceManager) InstanceExistsByProviderID(ctx context.Context, providerID string) (bool, error) {
	instanceID, err := instanceIDFromProviderID(providerID)
	if err != nil {
		return false, err
	}

	_, err = i.InstanceManager().GetByName(i.GetClient(), instanceID)
	if err != nil {
		if IsNotFoundError(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// nstanceShutdownByProviderID returns true if the instances is in safe state to detach volumes
func (i *InstanceManager) InstanceShutdownByProviderID(ctx context.Context, providerID string) (bool, error) {
	instanceID, err := instanceIDFromProviderID(providerID)
	if err != nil {
		return false, err
	}

	//host, err = getHostByName(i.GetClient(), instanceID)
	//_, err = getHostByName(i.GetClient(), instanceID)
	_, err = i.InstanceManager().GetByName(i.GetClient(), instanceID)
	if err != nil {
		return false, err
	}

	/*if host.HostStatus == "online" {
		return true, nil
	}*/
	return true, nil
}

// NodeAddresses implements Instances.NodeAddresses
func (i *InstanceManager) NodeAddresses(ctx context.Context, name types.NodeName) ([]v1.NodeAddress, error) {
	addrs, err := GetInstanceAddressByName(i.InstanceManager(), i.GetClient(), string(name))
	if err != nil {
		return nil, err
	}
	return addrs, nil
}

// NodeAddressesByProviderID returns the node addresses of an instances with the specified unique providerID
// This method will not be called from the node that is requesting this ID. i.e. metadata service
// and other local methods cannot be used here
func (i *InstanceManager) NodeAddressesByProviderID(ctx context.Context, providerID string) ([]v1.NodeAddress, error) {
	instanceID, err := instanceIDFromProviderID(providerID)
	if err != nil {
		return []v1.NodeAddress{}, err
	}

	return GetInstanceAddressByName(i.InstanceManager(), i.GetClient(), instanceID)
}

// InstanceID returns the cloud provider ID of the specified instance
func (i *InstanceManager) InstanceID(ctx context.Context, name types.NodeName) (string, error) {
	log.Infof("Get InstanceID by node name: %s", name)
	ins, err := i.InstanceManager().GetByName(i.GetClient(), string(name))
	if err != nil {
		if IsNotFoundError(err) {
			return "", cloudprovider.InstanceNotFound
		}
		return "", err
	}
	id := fmt.Sprintf("/%s", ins.GetId())
	log.Infof("Generate InstanceID %s", id)
	return id, nil
}

// InstanceTypeByProviderID returns the cloudprovider instance type of the node with the specified unique providerID
// This method will not be called from the node that is requesting this ID. i.e. metadata service
// and other local methods cannot be used here
func (i *InstanceManager) InstanceTypeByProviderID(ctx context.Context, providerID string) (string, error) {
	instanceID, err := instanceIDFromProviderID(providerID)
	if err != nil {
		return "", err
	}

	ins, err := i.InstanceManager().GetByName(i.GetClient(), instanceID)
	if err != nil {
		return "", err
	}

	return generateInstanceType(ins), nil
}

// InstanceType returns the type of the specified instance
func (i *InstanceManager) InstanceType(ctx context.Context, name types.NodeName) (string, error) {
	ins, err := i.InstanceManager().GetByName(i.GetClient(), string(name))
	if err != nil {
		return "", err
	}

	return generateInstanceType(ins), nil
}

// instanceIDFromProviderID splits a provider's id and return instanceID.
// A providerID is build out of '${ProviderName}:///${instance-id}' which contains ':///'.
// See cloudprovider.GetInstanceProviderID and Instances.InstanceID
func instanceIDFromProviderID(providerID string) (isntanceID string, err error) {
	// if Instances.InstanceID or cloudprovider.GetInstanceProviderID is changed, the regexp should be changed too
	log.Infof("instanceIDFromProviderID providerID: %q", providerID)
	var providerIDRegexp = regexp.MustCompile(`^` + ProviderName + `:///(.*)$`)
	matches := providerIDRegexp.FindStringSubmatch(providerID)
	var mErr1 error
	var mErr2 error
	if len(matches) != 2 {
		mErr1 = fmt.Errorf("ProviderID %q didn't match expected format \"yunion:///InstanceID\"", providerID)
	} else {
		return matches[1], nil
	}
	matches2 := strings.Split(providerID, "/")
	if len(matches2) != 2 {
		mErr2 = fmt.Errorf("Old providerID %q didn't match expected format \"/InstanceID\"", providerID)
	} else {
		return matches2[1], nil
	}
	return "", fmt.Errorf("Error1: %v, Error2: %v", mErr1, mErr2)
}

func generateInstanceType(ins drivers.Instance) string {
	specs := []string{
		fmt.Sprintf("%dC", ins.GetCPUCount()),
		fmt.Sprintf("%dG", ins.GetMemorySize()/1024),
		//fmt.Sprintf("type:%s", host.Type)
	}
	return strings.Join(specs, "")
}
