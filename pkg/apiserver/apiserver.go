// Copyright 2022 Antrea Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package apiserver

import (
	controllers "antrea.io/antreacloud/pkg/controllers/cloud"
	"context"
	"net"

	logger "github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	genericoptions "k8s.io/apiserver/pkg/server/options"
	controllerruntime "sigs.k8s.io/controller-runtime"

	controlplane "antrea.io/antreacloud/apis/controlplane/v1alpha1"
	"antrea.io/antreacloud/pkg/apiserver/registry/networkinterface"
	"antrea.io/antreacloud/pkg/apiserver/registry/virtualmachine"
)

var (
	// APIService listening port number.
	apiServerPort = 5443
	// Match Antrea Cloud Service Name
	antreaCloudSvcName = "antreacloud-cloud-service"
	// Match Antrea Cloud Service Domain Name
	antreaCloudDomainName = "antreacloud-cloud-service.antreacloud-system.svc"
)

// ExtraConfig holds custom apiserver config.
type ExtraConfig struct {
	// Place you custom config here.
	cloudInventory *controllers.CloudInventory
}

// Config defines the config for the apiserver.
type Config struct {
	GenericConfig *genericapiserver.RecommendedConfig
	ExtraConfig   ExtraConfig
}

func NewConfig(codecs serializer.CodecFactory, inventory *controllers.CloudInventory) (*Config, error) {
	recommend := genericoptions.NewRecommendedOptions("", nil)
	serverConfig := genericapiserver.NewRecommendedConfig(codecs)
	recommend.SecureServing.BindPort = apiServerPort

	// tls.crt and tls.key is populated by cert-manager injector.
	recommend.SecureServing.ServerCert.PairName = "tls"
	recommend.SecureServing.ServerCert.CertDirectory = "/tmp/k8s-apiserver/serving-certs"
	if err := recommend.SecureServing.MaybeDefaultWithSelfSignedCerts(antreaCloudSvcName,
		[]string{antreaCloudDomainName}, []net.IP{net.ParseIP("127.0.0.1")}); err != nil {
		return nil, err
	}

	if err := recommend.SecureServing.ApplyTo(&serverConfig.SecureServing, &serverConfig.LoopbackClientConfig); err != nil {
		return nil, err
	}
	if err := recommend.Authentication.ApplyTo(&serverConfig.Authentication, serverConfig.SecureServing,
		serverConfig.OpenAPIConfig); err != nil {
		return nil, err
	}
	if err := recommend.Authorization.ApplyTo(&serverConfig.Authorization); err != nil {
		return nil, err
	}
	config := &Config{
		GenericConfig: serverConfig,
		ExtraConfig: ExtraConfig{
			cloudInventory: inventory,
		},
	}
	return config, nil
}

// CloudControllerAPIServer contains state for a Kubernetes cluster master/api server.
type CloudControllerAPIServer struct {
	genericAPIServer *genericapiserver.GenericAPIServer
	logger           logger.Logger
}

func (s *CloudControllerAPIServer) Start(stop context.Context) error {
	s.logger.Info("Starting APIServer")
	err := s.genericAPIServer.PrepareRun().Run(stop.Done())
	if err != nil {
		s.logger.Error(err, "Failed to run APIServer")
	}
	return err
}

func (s *CloudControllerAPIServer) SetupWithManager(
	mgr controllerruntime.Manager,
	inventory *controllers.CloudInventory,
	logger logger.Logger) error {
	s.logger = logger
	codecs := serializer.NewCodecFactory(mgr.GetScheme())
	apiConfig, err := NewConfig(codecs, inventory)
	if err != nil {
		s.logger.Error(err, "unable to create APIServer config")
		return err
	}

	s.genericAPIServer, err = apiConfig.Complete().New(mgr.GetScheme(), codecs, s.logger)
	if err != nil {
		s.logger.Error(err, "unable to create APIServer")
		return err
	}
	if err = mgr.Add(s); err != nil {
		return err
	}
	return nil
}

type completedConfig struct {
	GenericConfig genericapiserver.CompletedConfig
	ExtraConfig   *ExtraConfig
}

// CompletedConfig embeds a private pointer that cannot be instantiated outside of this package.
type CompletedConfig struct {
	*completedConfig
}

// Complete fills in any fields not set that are required to have valid data. It's mutating the receiver.
func (cfg *Config) Complete() CompletedConfig {
	c := completedConfig{
		cfg.GenericConfig.Complete(),
		&cfg.ExtraConfig,
	}
	c.GenericConfig.Version = &version.Info{
		Major: "1",
		Minor: "0",
	}
	return CompletedConfig{&c}
}

// New returns a new instance of CloudControllerAPIServer from the given config.
func (c completedConfig) New(scheme *runtime.Scheme, codecs serializer.CodecFactory,
	logger logger.Logger) (*genericapiserver.GenericAPIServer, error) {
	genericServer, err := c.GenericConfig.New("cloud-controller-apiserver", genericapiserver.NewEmptyDelegate())
	if err != nil {
		return nil, err
	}

	virtualMachineStorage := virtualmachine.NewREST(c.ExtraConfig.cloudInventory, logger.WithName("VirtualMachine"))
	networkInterfaceStorage := networkinterface.NewREST(c.ExtraConfig.cloudInventory, logger.WithName("NetworkInterface"))

	cpGroup := genericapiserver.NewDefaultAPIGroupInfo(controlplane.GroupVersion.Group, scheme, metav1.ParameterCodec, codecs)
	cpv1alpha1Storage := map[string]rest.Storage{}
	cpv1alpha1Storage["virtualmachines"] = virtualMachineStorage
	cpv1alpha1Storage["networkinterfaces"] = networkInterfaceStorage

	cpGroup.VersionedResourcesStorageMap["v1alpha1"] = cpv1alpha1Storage

	if err := genericServer.InstallAPIGroup(&cpGroup); err != nil {
		return nil, err
	}
	return genericServer, nil
}
