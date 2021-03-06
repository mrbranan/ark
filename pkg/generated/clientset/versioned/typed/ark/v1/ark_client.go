/*
Copyright 2018 the Heptio Ark contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by client-gen. DO NOT EDIT.

package v1

import (
	v1 "github.com/heptio/ark/pkg/apis/ark/v1"
	"github.com/heptio/ark/pkg/generated/clientset/versioned/scheme"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer"
	rest "k8s.io/client-go/rest"
)

type ArkV1Interface interface {
	RESTClient() rest.Interface
	BackupsGetter
	BackupStorageLocationsGetter
	ConfigsGetter
	DeleteBackupRequestsGetter
	DownloadRequestsGetter
	PodVolumeBackupsGetter
	PodVolumeRestoresGetter
	ResticRepositoriesGetter
	RestoresGetter
	SchedulesGetter
}

// ArkV1Client is used to interact with features provided by the ark.heptio.com group.
type ArkV1Client struct {
	restClient rest.Interface
}

func (c *ArkV1Client) Backups(namespace string) BackupInterface {
	return newBackups(c, namespace)
}

func (c *ArkV1Client) BackupStorageLocations(namespace string) BackupStorageLocationInterface {
	return newBackupStorageLocations(c, namespace)
}

func (c *ArkV1Client) Configs(namespace string) ConfigInterface {
	return newConfigs(c, namespace)
}

func (c *ArkV1Client) DeleteBackupRequests(namespace string) DeleteBackupRequestInterface {
	return newDeleteBackupRequests(c, namespace)
}

func (c *ArkV1Client) DownloadRequests(namespace string) DownloadRequestInterface {
	return newDownloadRequests(c, namespace)
}

func (c *ArkV1Client) PodVolumeBackups(namespace string) PodVolumeBackupInterface {
	return newPodVolumeBackups(c, namespace)
}

func (c *ArkV1Client) PodVolumeRestores(namespace string) PodVolumeRestoreInterface {
	return newPodVolumeRestores(c, namespace)
}

func (c *ArkV1Client) ResticRepositories(namespace string) ResticRepositoryInterface {
	return newResticRepositories(c, namespace)
}

func (c *ArkV1Client) Restores(namespace string) RestoreInterface {
	return newRestores(c, namespace)
}

func (c *ArkV1Client) Schedules(namespace string) ScheduleInterface {
	return newSchedules(c, namespace)
}

// NewForConfig creates a new ArkV1Client for the given config.
func NewForConfig(c *rest.Config) (*ArkV1Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &ArkV1Client{client}, nil
}

// NewForConfigOrDie creates a new ArkV1Client for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *ArkV1Client {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new ArkV1Client for the given RESTClient.
func New(c rest.Interface) *ArkV1Client {
	return &ArkV1Client{c}
}

func setConfigDefaults(config *rest.Config) error {
	gv := v1.SchemeGroupVersion
	config.GroupVersion = &gv
	config.APIPath = "/apis"
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: scheme.Codecs}

	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	return nil
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *ArkV1Client) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}
