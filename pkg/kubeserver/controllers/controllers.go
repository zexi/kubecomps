package controllers

import (
	"fmt"

	"k8s.io/client-go/kubernetes"

	"yunion.io/x/kubecomps/pkg/kubeserver/controllers/auth"
	"yunion.io/x/kubecomps/pkg/kubeserver/controllers/helm"
)

var Manager *SControllerManager

func init() {
	Manager = newControllerManager()
}

func Start() {
	helm.Start()
}

type SControllerManager struct {
	controllerMap map[string]*SClusterController
}

func newControllerManager() *SControllerManager {
	return &SControllerManager{
		controllerMap: make(map[string]*SClusterController),
	}
}

func (m *SControllerManager) GetController(clusterId string) (*SClusterController, error) {
	ctrl, ok := m.controllerMap[clusterId]
	if !ok {
		return nil, fmt.Errorf("Cluster controller %q not found", clusterId)
	}
	return ctrl, nil
}

type SClusterController struct {
	clusterId             string
	clusterName           string
	keystoneAuthenticator *auth.KeystoneAuthenticator
	stopCh                chan struct{}
}

func (c *SClusterController) RunKeystoneAuthenticator(k8sCli *kubernetes.Clientset, stopCh chan struct{}) {
	c.keystoneAuthenticator = auth.NewKeystoneAuthenticator(k8sCli, stopCh)
}

func (c *SClusterController) GetKeystoneAuthenticator() *auth.KeystoneAuthenticator {
	return c.keystoneAuthenticator
}
