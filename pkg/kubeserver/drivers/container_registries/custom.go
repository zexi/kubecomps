package container_registries

import (
	"context"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/drivers/container_registries/client"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/pkg/errors"
)

func init() {
	models.RegisterContainerRegistryDriver(newCustomImpl())
}

func newCustomImpl() models.IContainerRegistryDriver {
	return new(customImpl)
}

type customImpl struct{}

func (c customImpl) GetType() api.ContainerRegistryType {
	return api.ContainerRegistryTypeCustom
}

func (c customImpl) DownloadImage(ctx context.Context, url string, conf *api.ContainerRegistryConfig, input api.ContainerRegistryDownloadImageInput) (string, error) {
	return "", httperrors.NewNotSupportedError("custom not support download image")
}

func (h customImpl) GetDockerRegistryClient(url string, config *api.ContainerRegistryConfig) (client.Client, error) {
	if config.Custom == nil {
		return nil, errors.Errorf("custom config is nil")
	}
	return client.NewClient(url, client.DockerAuthConfig{
		Username: config.Custom.Username,
		Password: config.Custom.Password,
	})
}
