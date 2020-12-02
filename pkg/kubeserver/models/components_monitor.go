package models

import (
	"fmt"

	"k8s.io/api/core/v1"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/embed"
	"yunion.io/x/kubecomps/pkg/kubeserver/templates/components"
	"yunion.io/x/kubecomps/pkg/kubeserver/utils/certificates"
)

var (
	MonitorComponentManager *SMonitorComponentManager
)

const (
	MonitorNamespace   = "onecloud-monitoring"
	MonitorReleaseName = "monitor"
)

func init() {
	MonitorComponentManager = NewMonitorComponentManager()
	ComponentManager.RegisterDriver(newComponentDriverMonitor())
}

type SMonitorComponentManager struct {
	SComponentManager
	HelmComponentManager
}

type SMonitorComponent struct {
	SComponent
}

func NewMonitorComponentManager() *SMonitorComponentManager {
	man := new(SMonitorComponentManager)
	man.SComponentManager = *NewComponentManager(SMonitorComponent{},
		"kubecomponentmonitor",
		"kubecomponentmonitors")
	man.HelmComponentManager = *NewHelmComponentManager(MonitorNamespace, MonitorReleaseName, embed.MONITOR_STACK_V1_TGZ)
	man.SetVirtualObject(man)
	return man
}

type componentDriverMonitor struct {
	baseComponentDriver
}

func newComponentDriverMonitor() IComponentDriver {
	return new(componentDriverMonitor)
}

func (c componentDriverMonitor) GetType() string {
	return api.ClusterComponentMonitor
}

func (c componentDriverMonitor) ValidateCreateData(userCred mcclient.TokenCredential, cluster *SCluster, input *api.ComponentCreateInput) error {
	return c.validateSetting(userCred, cluster, input.Monitor)
}

func (c componentDriverMonitor) validateSetting(userCred mcclient.TokenCredential, cluster *SCluster, conf *api.ComponentSettingMonitor) error {
	if conf == nil {
		return httperrors.NewInputParameterError("monitor config is empty")
	}
	if err := c.validateGrafana(userCred, cluster, conf.Grafana); err != nil {
		return errors.Wrap(err, "component grafana")
	}
	if err := c.validateLoki(userCred, cluster, conf.Loki); err != nil {
		return errors.Wrap(err, "component loki")
	}
	if err := c.validatePrometheus(userCred, cluster, conf.Prometheus); err != nil {
		return errors.Wrap(err, "component prometheus")
	}
	if err := c.validatePromtail(conf.Promtail); err != nil {
		return errors.Wrap(err, "component promtail")
	}
	return nil
}

func (c componentDriverMonitor) validateGrafana(userCred mcclient.TokenCredential, cluster *SCluster, conf *api.ComponentSettingMonitorGrafana) error {
	if conf.Storage.Enabled {
		if err := c.validateStorage(userCred, cluster, conf.Storage); err != nil {
			return err
		}
	}
	if conf.Host == "" && conf.PublicAddress == "" {
		return httperrors.NewInputParameterError("grafana public address or host must provide")
	}
	if conf.TLSKeyPair == nil {
		// return httperrors.NewInputParameterError("grafana tls key pair must provide")
		kp, err := certificates.GetOrGenerateCACert(nil, "grafana-tls")
		if err != nil {
			return errors.Wrap(err, "generate grafana tls keypair")
		}
		conf.TLSKeyPair = &api.TLSKeyPair{
			Certificate: string(kp.Cert),
			Key:         string(kp.Key),
		}
	}
	if err := c.validateGrafanaTLSKeyPair(conf.TLSKeyPair); err != nil {
		return errors.Wrap(err, "validate tls key pair")
	}
	return nil
}

func (c componentDriverMonitor) validateGrafanaTLSKeyPair(pair *api.TLSKeyPair) error {
	if pair.Certificate == "" {
		return httperrors.NewInputParameterError("tls certificate not provide")
	}
	if pair.Key == "" {
		return httperrors.NewInputParameterError("tls key not provide")
	}
	if pair.Name == "" {
		pair.Name = "grafana-ingress-tls"
	}
	return nil
}

func (c componentDriverMonitor) validateLoki(userCred mcclient.TokenCredential, cluster *SCluster, conf *api.ComponentSettingMonitorLoki) error {
	if conf.Storage.Enabled {
		if err := c.validateStorage(userCred, cluster, conf.Storage); err != nil {
			return err
		}
	}
	return nil
}

func (c componentDriverMonitor) validateStorage(userCred mcclient.TokenCredential, cluster *SCluster, storage *api.ComponentStorage) error {
	if storage == nil {
		return httperrors.NewInputParameterError("storage config is empty")
	}
	if storage.SizeMB < 1024 {
		return httperrors.NewNotAcceptableError("storage size must large than 1 GB")
	}
	if storage.ClassName != "" {
		obj, err := GetStorageClassManager().GetByIdOrName(userCred, cluster.GetId(), storage.ClassName)
		if err != nil {
			return errors.Wrapf(err, "get storageClass %s", storage.ClassName)
		}
		storage.ClassName = obj.GetName()
	}
	return nil
}

func (c componentDriverMonitor) validatePrometheus(userCred mcclient.TokenCredential, cluster *SCluster, conf *api.ComponentSettingMonitorPrometheus) error {
	if conf == nil {
		return httperrors.NewInputParameterError("config is empty")
	}
	if conf.Storage.Enabled {
		if err := c.validateStorage(userCred, cluster, conf.Storage); err != nil {
			return err
		}
	}
	return nil
}

func (c componentDriverMonitor) validatePromtail(conf *api.ComponentSettingMonitorPromtail) error {
	// TODO
	return nil
}

func (c componentDriverMonitor) ValidateUpdateData(userCred mcclient.TokenCredential, cluster *SCluster, input *api.ComponentUpdateInput) error {
	return c.validateSetting(userCred, cluster, input.Monitor)
}

func (c componentDriverMonitor) GetCreateSettings(input *api.ComponentCreateInput) (*api.ComponentSettings, error) {
	if input.ComponentSettings.Namespace == "" {
		input.ComponentSettings.Namespace = MonitorNamespace
	}
	return &input.ComponentSettings, nil
}

func (c componentDriverMonitor) GetUpdateSettings(oldSetting *api.ComponentSettings, input *api.ComponentUpdateInput) (*api.ComponentSettings, error) {
	oldSetting.Monitor = input.Monitor
	return oldSetting, nil
}

func (c componentDriverMonitor) DoEnable(cluster *SCluster, setting *api.ComponentSettings) error {
	return MonitorComponentManager.CreateHelmResource(cluster, setting)
}

func (c componentDriverMonitor) DoDisable(cluster *SCluster, setting *api.ComponentSettings) error {
	return MonitorComponentManager.DeleteHelmResource(cluster, setting)
}

func (c componentDriverMonitor) DoUpdate(cluster *SCluster, setting *api.ComponentSettings) error {
	return MonitorComponentManager.UpdateHelmResource(cluster, setting)
}

func (c componentDriverMonitor) FetchStatus(cluster *SCluster, comp *SComponent, status *api.ComponentsStatus) error {
	if status.Monitor == nil {
		status.Monitor = new(api.ComponentStatusMonitor)
	}
	c.InitStatus(comp, &status.Monitor.ComponentStatus)
	return nil
}

func (m SMonitorComponentManager) getHelmValues(cluster *SCluster, setting *api.ComponentSettings) (map[string]interface{}, error) {
	imgRepo, err := cluster.GetImageRepository()
	if err != nil {
		return nil, errors.Wrapf(err, "get cluster %s repo", cluster.GetName())
	}
	input := setting.Monitor
	if input.Grafana.AdminUser == "" {
		input.Grafana.AdminUser = "admin"
	}
	if input.Grafana.AdminPassword == "" {
		input.Grafana.AdminPassword = "prom-operator"
	}
	repo := imgRepo.Url
	mi := func(name, tag string) components.Image {
		return components.Image{
			Repository: fmt.Sprintf("%s/%s", repo, name),
			Tag:        tag,
		}
	}
	grafanaHost := input.Grafana.Host
	if grafanaHost == "" {
		grafanaHost = input.Grafana.PublicAddress
	}
	grafanaProto := "https"
	if input.Grafana.TLSKeyPair == nil {
		return nil, errors.Errorf("grafana tls key pair not provided")
	}
	grafanaIngressTLS := []*api.IngressTLS{
		{
			SecretName: input.Grafana.TLSKeyPair.Name,
		},
	}
	conf := components.MonitorStack{
		Prometheus: components.Prometheus{
			Spec: components.PrometheusSpec{
				Image: mi("prometheus", "v2.15.2"),
			},
		},
		Alertmanager: components.Alertmanager{
			Spec: components.AlertmanagerSpec{
				Image: mi("alertmanager", "v0.20.0"),
			},
		},
		PrometheusNodeExporter: components.PrometheusNodeExporter{
			Image: mi("node-exporter", "v0.18.1"),
		},
		KubeStateMetrics: components.KubeStateMetrics{
			Image: mi("kube-state-metrics", "v1.9.4"),
		},
		Grafana: components.Grafana{
			AdminUser:     input.Grafana.AdminUser,
			AdminPassword: input.Grafana.AdminPassword,
			Sidecar: components.GrafanaSidecar{
				Image: mi("k8s-sidecar", "0.1.99"),
			},
			Image: mi("grafana", "6.7.1"),
			Service: &components.Service{
				Type: string(v1.ServiceTypeClusterIP),
			},
			Ingress: &components.GrafanaIngress{
				Enabled: true,
				Path:    "/grafana",
				Host:    input.Grafana.Host,
				Secret:  input.Grafana.TLSKeyPair,
				TLS:     grafanaIngressTLS,
			},
			GrafanaIni: &components.GrafanaIni{
				Server: &components.GrafanaIniServer{
					ServeFromSubPath: true,
					RootUrl:          fmt.Sprintf("%s://%s/grafana/", grafanaProto, grafanaHost),
				},
			},
		},
		Loki: components.Loki{
			Image: mi("loki", "1.4.1"),
		},
		Promtail: components.Promtail{
			Image: mi("promtail", "1.4.1"),
		},
		PrometheusOperator: components.PrometheusOperator{
			Image:                         mi("prometheus-operator", "v0.37.0"),
			ConfigmapReloadImage:          mi("configmap-reload", "v0.0.1"),
			PrometheusConfigReloaderImage: mi("prometheus-config-reloader", "v0.37.0"),
			TLSProxy: components.PromTLSProxy{
				Image: mi("ghostunnel", "v1.5.2"),
			},
			AdmissionWebhooks: components.AdmissionWebhooks{
				Enabled: false,
				Patch: components.AdmissionWebhooksPatch{
					Enabled: false,
					Image:   mi("kube-webhook-certgen", "v1.0.0"),
				},
			},
		},
	}
	if input.Prometheus.Storage != nil && input.Prometheus.Storage.Enabled {
		spec, err := components.NewPrometheusStorageSpec(*input.Prometheus.Storage)
		if err != nil {
			return nil, errors.Wrap(err, "prometheus storage spec")
		}
		conf.Prometheus.Spec.StorageSpec = spec
	}
	if input.Grafana.Storage != nil && input.Grafana.Storage.Enabled {
		spec, err := components.NewPVCStorage(input.Grafana.Storage)
		if err != nil {
			return nil, errors.Wrap(err, "grafana storage spec")
		}
		conf.Grafana.Storage = spec
	}
	if input.Loki.Storage != nil && input.Loki.Storage.Enabled {
		spec, err := components.NewPVCStorage(input.Loki.Storage)
		if err != nil {
			return nil, errors.Wrap(err, "loki storage")
		}
		conf.Loki.Storage = spec
	}
	return components.GenerateHelmValues(conf), nil
}

func (m SMonitorComponentManager) CreateHelmResource(cluster *SCluster, setting *api.ComponentSettings) error {
	vals, err := m.getHelmValues(cluster, setting)
	if err != nil {
		return errors.Wrap(err, "get helm config values")
	}
	return m.HelmComponentManager.CreateHelmResource(cluster, vals)
}

func (m SMonitorComponentManager) DeleteHelmResource(cluster *SCluster, setting *api.ComponentSettings) error {
	return m.HelmComponentManager.DeleteHelmResource(cluster)
}

func (m SMonitorComponentManager) UpdateHelmResource(cluster *SCluster, setting *api.ComponentSettings) error {
	vals, err := m.getHelmValues(cluster, setting)
	if err != nil {
		return errors.Wrap(err, "get helm config values")
	}
	return m.HelmComponentManager.UpdateHelmResource(cluster, vals)
}
