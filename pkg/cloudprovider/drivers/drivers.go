package drivers

import (
	"fmt"
	"sync"
)

type driversManager struct {
	*sync.Map
}

func newDriversManager() *driversManager {
	return &driversManager{
		Map: new(sync.Map),
	}
}

func (m driversManager) registerDriver(drv resourceManager) {
	m.Store(drv.GetDriverName(), drv)
}

func (m driversManager) getDriver(name string) (resourceManager, error) {
	drv, ok := m.Load(name)
	if !ok {
		return nil, fmt.Errorf("Not found driver %s", name)
	}
	return drv.(resourceManager), nil
}

type instancesManager struct {
	*driversManager
}

func newInstancesManager() *instancesManager {
	return &instancesManager{driversManager: newDriversManager()}
}

func (m instancesManager) getDriver(name string) (InstanceManager, error) {
	drv, err := m.driversManager.getDriver(name)
	if err != nil {
		return nil, err
	}
	return drv.(InstanceManager), nil
}

var (
	instanceManagers *instancesManager
)

func init() {
	instanceManagers = newInstancesManager()
}

func RegisterInstanceDriver(drv InstanceManager) {
	instanceManagers.registerDriver(drv)
}

func GetInstanceManager(instanceType string) (InstanceManager, error) {
	return instanceManagers.getDriver(instanceType)
}
