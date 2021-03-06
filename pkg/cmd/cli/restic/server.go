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
package restic

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	corev1informers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/heptio/ark/pkg/buildinfo"
	"github.com/heptio/ark/pkg/client"
	"github.com/heptio/ark/pkg/cloudprovider/azure"
	"github.com/heptio/ark/pkg/cmd"
	"github.com/heptio/ark/pkg/cmd/util/signals"
	"github.com/heptio/ark/pkg/controller"
	clientset "github.com/heptio/ark/pkg/generated/clientset/versioned"
	informers "github.com/heptio/ark/pkg/generated/informers/externalversions"
	"github.com/heptio/ark/pkg/restic"
	"github.com/heptio/ark/pkg/util/logging"
)

func NewServerCommand(f client.Factory) *cobra.Command {
	var (
		logLevelFlag = logging.LogLevelFlag(logrus.InfoLevel)
		location     = "default"
	)

	var command = &cobra.Command{
		Use:   "server",
		Short: "Run the ark restic server",
		Long:  "Run the ark restic server",
		Run: func(c *cobra.Command, args []string) {
			logLevel := logLevelFlag.Parse()
			logrus.Infof("Setting log-level to %s", strings.ToUpper(logLevel.String()))

			logger := logging.DefaultLogger(logLevel)
			logger.Infof("Starting Ark restic server %s", buildinfo.FormattedGitSHA())

			s, err := newResticServer(logger, fmt.Sprintf("%s-%s", c.Parent().Name(), c.Name()), location)
			cmd.CheckError(err)

			s.run()
		},
	}

	command.Flags().Var(logLevelFlag, "log-level", fmt.Sprintf("the level at which to log. Valid values are %s.", strings.Join(logLevelFlag.AllowedValues(), ", ")))
	command.Flags().StringVar(&location, "default-backup-storage-location", location, "name of the default backup storage location")

	return command
}

type resticServer struct {
	kubeClient          kubernetes.Interface
	arkClient           clientset.Interface
	arkInformerFactory  informers.SharedInformerFactory
	kubeInformerFactory kubeinformers.SharedInformerFactory
	podInformer         cache.SharedIndexInformer
	secretInformer      cache.SharedIndexInformer
	logger              logrus.FieldLogger
	ctx                 context.Context
	cancelFunc          context.CancelFunc
}

func newResticServer(logger logrus.FieldLogger, baseName, locationName string) (*resticServer, error) {
	clientConfig, err := client.Config("", "", baseName)
	if err != nil {
		return nil, err
	}

	kubeClient, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	arkClient, err := clientset.NewForConfig(clientConfig)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	location, err := arkClient.ArkV1().BackupStorageLocations(os.Getenv("HEPTIO_ARK_NAMESPACE")).Get(locationName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if location.Spec.Provider == "azure" {
		if err := azure.SetResticEnvVars(location.Spec.Config); err != nil {
			return nil, err
		}
	}

	// use a stand-alone pod informer because we want to use a field selector to
	// filter to only pods scheduled on this node.
	podInformer := corev1informers.NewFilteredPodInformer(
		kubeClient,
		metav1.NamespaceAll,
		0,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
		func(opts *metav1.ListOptions) {
			opts.FieldSelector = fmt.Sprintf("spec.nodeName=%s", os.Getenv("NODE_NAME"))
		},
	)

	// use a stand-alone secrets informer so we can filter to only the restic credentials
	// secret(s) within the heptio-ark namespace
	//
	// note: using an informer to access the single secret for all ark-managed
	// restic repositories is overkill for now, but will be useful when we move
	// to fully-encrypted backups and have unique keys per repository.
	secretInformer := corev1informers.NewFilteredSecretInformer(
		kubeClient,
		os.Getenv("HEPTIO_ARK_NAMESPACE"),
		0,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
		func(opts *metav1.ListOptions) {
			opts.FieldSelector = fmt.Sprintf("metadata.name=%s", restic.CredentialsSecretName)
		},
	)

	ctx, cancelFunc := context.WithCancel(context.Background())

	return &resticServer{
		kubeClient:          kubeClient,
		arkClient:           arkClient,
		arkInformerFactory:  informers.NewFilteredSharedInformerFactory(arkClient, 0, os.Getenv("HEPTIO_ARK_NAMESPACE"), nil),
		kubeInformerFactory: kubeinformers.NewSharedInformerFactory(kubeClient, 0),
		podInformer:         podInformer,
		secretInformer:      secretInformer,
		logger:              logger,
		ctx:                 ctx,
		cancelFunc:          cancelFunc,
	}, nil
}

func (s *resticServer) run() {
	signals.CancelOnShutdown(s.cancelFunc, s.logger)

	s.logger.Info("Starting controllers")

	var wg sync.WaitGroup

	backupController := controller.NewPodVolumeBackupController(
		s.logger,
		s.arkInformerFactory.Ark().V1().PodVolumeBackups(),
		s.arkClient.ArkV1(),
		s.podInformer,
		s.secretInformer,
		s.kubeInformerFactory.Core().V1().PersistentVolumeClaims(),
		os.Getenv("NODE_NAME"),
	)
	wg.Add(1)
	go func() {
		defer wg.Done()
		backupController.Run(s.ctx, 1)
	}()

	restoreController := controller.NewPodVolumeRestoreController(
		s.logger,
		s.arkInformerFactory.Ark().V1().PodVolumeRestores(),
		s.arkClient.ArkV1(),
		s.podInformer,
		s.secretInformer,
		s.kubeInformerFactory.Core().V1().PersistentVolumeClaims(),
		os.Getenv("NODE_NAME"),
	)
	wg.Add(1)
	go func() {
		defer wg.Done()
		restoreController.Run(s.ctx, 1)
	}()

	go s.arkInformerFactory.Start(s.ctx.Done())
	go s.kubeInformerFactory.Start(s.ctx.Done())
	go s.podInformer.Run(s.ctx.Done())
	go s.secretInformer.Run(s.ctx.Done())

	s.logger.Info("Controllers started successfully")

	<-s.ctx.Done()

	s.logger.Info("Waiting for all controllers to shut down gracefully")
	wg.Wait()
}
