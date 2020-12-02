package components

import (
	"fmt"

	"sigs.k8s.io/yaml"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/httperrors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
)

type Image struct {
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
}

type PromTLSProxy struct {
	Enabled bool  `json:"enabled"`
	Image   Image `json:"image"`
}

type PromAdmissionWebhooksPatch struct {
	Enabled bool `json:"enabled"`
}

type PromAdmissionWebhooks struct {
	Enabled bool                       `json:"enabled"`
	Patch   PromAdmissionWebhooksPatch `json:"patch"`
}

type Prometheus struct {
	Spec PrometheusSpec `json:"prometheusSpec"`
}

type Resources struct {
	Requests map[string]string `json:"requests"`
}

type PersistentVolumeClaimSpec struct {
	StorageClassName *string   `json:"storageClassName"`
	AccessModes      []string  `json:"accessModes"`
	Resources        Resources `json:"resources"`
}

type PersistentVolumeClaim struct {
	Spec PersistentVolumeClaimSpec `json:"spec"`
}

type PrometheusStorageSpec struct {
	Template PersistentVolumeClaim `json:"volumeClaimTemplate"`
}

func NewPrometheusStorageSpec(storage api.ComponentStorage) (*PrometheusStorageSpec, error) {
	sizeGB := storage.SizeMB / 1024
	if sizeGB <= 0 {
		return nil, httperrors.NewInputParameterError("size must large than 1GB")
	}
	storageSize := fmt.Sprintf("%dGi", sizeGB)
	spec := new(PersistentVolumeClaimSpec)
	if storage.ClassName != "" {
		spec.StorageClassName = &storage.ClassName
	}
	spec.AccessModes = storage.GetAccessModes()
	spec.Resources = Resources{
		Requests: map[string]string{
			"storage": storageSize,
		},
	}
	return &PrometheusStorageSpec{
		Template: PersistentVolumeClaim{
			Spec: *spec,
		},
	}, nil
}

type PrometheusSpec struct {
	// image: quay.io/prometheus/prometheus:v2.15.2
	Image       Image                  `json:"image"`
	StorageSpec *PrometheusStorageSpec `json:"storageSpec"`
}

type Alertmanager struct {
	Spec AlertmanagerSpec `json:"alertmanagerSpec"`
}

type AlertmanagerSpec struct {
	// image: quay.io/prometheus/alertmanager:v0.20.0
	Image Image `json:"image"`
}

type PrometheusNodeExporter struct {
	// image: quay.io/prometheus/node-exporter:v0.18.1
	Image Image `json:"image"`
}

type KubeStateMetrics struct {
	// image: quay.io/coreos/kube-state-metrics:v1.9.4
	Image Image `json:"image"`
}

type GrafanaSidecar struct {
	// image: kiwigrid/k8s-sidecar:0.1.99
	Image Image `json:"image"`
}

type Storage struct {
	Type             string   `json:"type"`
	Enabled          bool     `json:"enabled"`
	StorageClassName string   `json:"storageClassName"`
	AccessModes      []string `json:"accessModes"`
	Size             string   `json:"size"`
}

func NewPVCStorage(storage *api.ComponentStorage) (*Storage, error) {
	sizeMB := storage.SizeMB
	sizeGB := sizeMB / 1024
	if sizeGB <= 0 {
		return nil, httperrors.NewInputParameterError("size must large than 1GB")
	}
	accessModes := storage.GetAccessModes()
	return &Storage{
		Type:             "pvc",
		Enabled:          true,
		StorageClassName: storage.ClassName,
		AccessModes:      accessModes,
		Size:             fmt.Sprintf("%dGi", sizeGB),
	}, nil
}

type Service struct {
	Type string `json:"type"`
}

type GrafanaIngress struct {
	Enabled bool            `json:"enabled"`
	Path    string          `json:"path"`
	Host    string          `json:"host,allowempty"`
	Secret  *api.TLSKeyPair `json:"secret"`

	TLS []*api.IngressTLS `json:"tls"`
}

type GrafanaIniServer struct {
	RootUrl          string `json:"root_url"`
	ServeFromSubPath bool   `json:"serve_from_sub_path"`
}

type GrafanaIni struct {
	Server *GrafanaIniServer `json:"server"`
}

type Grafana struct {
	AdminUser     string         `json:"adminUser"`
	AdminPassword string         `json:"adminPassword"`
	Sidecar       GrafanaSidecar `json:"sidecar"`
	// image: grafana/grafana:6.7.1
	Image      Image           `json:"image"`
	Storage    *Storage        `json:"persistence"`
	Service    *Service        `json:"service"`
	Ingress    *GrafanaIngress `json:"ingress"`
	GrafanaIni *GrafanaIni     `json:"grafana.ini"`
}

type Loki struct {
	Image   Image    `json:"image"`
	Storage *Storage `json:"persistence"`
}

type Promtail struct {
	BusyboxImage string `json:"busyboxImage"`
	Image        Image  `json:"image"`
}

type AdmissionWebhooksPatch struct {
	Enabled bool `json:"enabled"`
	// image: jettech/kube-webhook-certgen:v1.0.0
	Image Image `json:"image"`
}

type AdmissionWebhooks struct {
	Enabled bool                   `json:"enabled"`
	Patch   AdmissionWebhooksPatch `json:"patch"`
}

type PrometheusOperator struct {
	// image: squareup/ghostunnel:v1.5.2
	TLSProxy          PromTLSProxy      `json:"tlsProxy"`
	AdmissionWebhooks AdmissionWebhooks `json:"admissionWebhooks"`
	// image: quay.io/coreos/prometheus-operator:v0.37.0
	Image Image `json:"image"`
	// image: quay.io/coreos/configmap-reload:v0.0.1
	ConfigmapReloadImage Image `json:"configmapReloadImage"`
	// image: quay.io/coreos/prometheus-config-reloader:v0.37.0
	PrometheusConfigReloaderImage Image `json:"prometheusConfigReloaderImage"`
	// image: k8s.gcr.io/hyperkube:v1.12.1
	HyperkubeImage Image `json:"hyperkubeImage"`
}

type MonitorStack struct {
	Prometheus             Prometheus             `json:"prometheus"`
	Alertmanager           Alertmanager           `json:"alertmanager"`
	PrometheusNodeExporter PrometheusNodeExporter `json:"prometheus-node-exporter"`
	KubeStateMetrics       KubeStateMetrics       `json:"kube-state-metrics"`
	Grafana                Grafana                `json:"grafana"`
	Loki                   Loki                   `json:"loki"`
	Promtail               Promtail               `json:"promtail"`
	PrometheusOperator     PrometheusOperator     `json:"prometheusOperator"`
}

func GenerateHelmValues(config interface{}) map[string]interface{} {
	yamlStr := jsonutils.Marshal(config).YAMLString()
	vals := map[string]interface{}{}
	yaml.Unmarshal([]byte(yamlStr), &vals)
	log.Errorf("====generate values: %s", yamlStr)
	return vals
}
