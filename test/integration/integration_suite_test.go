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

package integration

import (
	"flag"
	"io/ioutil"
	"math/rand"
	"path"
	"strings"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	antreatypes "antrea.io/antrea/pkg/apis/crd/v1alpha2"

	cloudv1alpha1 "antrea.io/antreacloud/apis/crd/v1alpha1"
	"antrea.io/antreacloud/pkg/logging"
	"antrea.io/antreacloud/test/utils"
)

const (
	focusCore           = "Core-test"
	focusAzureAgentless = "Extended-azure-agentless"
	focusAgentEks       = "Extended-test-agent-eks"
	focusAgentAks       = "Extended-test-agent-aks"
)

var (
	kubeCtl       *utils.KubeCtl
	k8sClient     client.Client
	k8sClients    map[string]client.Client
	cloudVPC      utils.CloudVPC
	cloudVPCs     map[string]utils.CloudVPC
	clusters      []string
	scheme        = runtime.NewScheme()
	preserveSetup = false
	testFocus     = []string{focusCore, focusAzureAgentless, focusAgentEks, focusAgentAks}
	cloudClusters = []string{focusAgentEks, focusAgentAks}
	cloudCluster  bool

	// flags.
	manifest            string
	preserveSetupOnFail bool
	supportBundleDir    string
	kubeconfig          string
	cloudProviders      string
	clusterContexts     string
)

func init() {
	flag.StringVar(&manifest, "manifest-path", "./config/antrea-cloud.yml", "The relative path to manifest.")
	flag.BoolVar(&preserveSetupOnFail, "preserve-setup-on-fail", false, "Preserve the setup if a test failed.")
	flag.StringVar(&supportBundleDir, "support-bundle-dir", "", "Support bundles are saved in this dir when specified")
	flag.StringVar(&cloudProviders, "cloud-provider", string(cloudv1alpha1.AWSCloudProvider),
		"cloud Providers to use, separated by comma. Default is aws")
	flag.StringVar(&clusterContexts, "cluster-context", "", "cluster context to use, separated by common. Default is empty")
	rand.Seed(time.Now().Unix())
}

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), logging.UseDevMode()))

	var err error

	By("Bootstrapping the test environment")
	err = clientgoscheme.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	err = cloudv1alpha1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	err = antreatypes.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	kubeconfig = flag.Lookup("kubeconfig").Value.(flag.Getter).Get().(string)
	kubeCtl, err = utils.NewKubeCtl(kubeconfig)
	Expect(err).ToNot(HaveOccurred())
	Expect(kubeCtl).ToNot(BeNil())

	antreaCloudManifests := make(map[string]string)
	k8sClients = make(map[string]client.Client)
	clusters = strings.Split(clusterContexts, ",")
	for _, cluster := range clusters {
		bytes, err := ioutil.ReadFile(manifest)
		Expect(err).ToNot(HaveOccurred())
		antreaCloudManifests[cluster] = string(bytes)

		c, err := utils.NewK8sClient(scheme, cluster)
		Expect(err).ToNot(HaveOccurred())
		Expect(c).ToNot(BeNil())
		k8sClients[cluster] = c
	}

	// Create VM VPC in parallel.
	wg := sync.WaitGroup{}
	wgChan := make(chan error)
	cloudVPCs = make(map[string]utils.CloudVPC)
	for _, provider := range strings.Split(cloudProviders, ",") {
		vpc, err := utils.NewCloudVPC(cloudv1alpha1.CloudProvider(provider))
		Expect(err).ToNot(HaveOccurred())
		cloudVPCs[provider] = vpc
		wg.Add(1)
		go func() {
			defer wg.Done()
			if vpc.IsConfigured() {
				return
			}
			err := vpc.Reapply(time.Second * 300)
			wgChan <- err
		}()
	}
	go func() {
		wg.Wait()
		close(wgChan)
	}()

	for _, cluster := range clusters {
		antreaCloudManifest := antreaCloudManifests[cluster]
		kubeCtl.SetContext(cluster)
		cl := k8sClients[cluster]
		if len(cluster) == 0 {
			cluster = "default"
		}
		By(cluster + ": Check cert-manager is ready, may wait longer for docker pull")
		// Increate the timeout for now to get past CI/CD timeout at this point to see what is causing it.
		err = utils.RestartOrWaitDeployment(cl, "cert-manager", "cert-manager", time.Second*240, false)
		Expect(err).ToNot(HaveOccurred())
		err = utils.RestartOrWaitDeployment(cl, "cert-manager-cainjector", "cert-manager", time.Second*120, false)
		Expect(err).ToNot(HaveOccurred())
		err = utils.RestartOrWaitDeployment(cl, "cert-manager-webhook", "cert-manager", time.Second*120, false)
		Expect(err).ToNot(HaveOccurred())

		By(cluster + ": Check antrea controller is ready, may wait longer for docker pull")
		err = utils.RestartOrWaitDeployment(cl, "antrea-controller", "kube-system", time.Second*120, false)
		Expect(err).ToNot(HaveOccurred())

		By(cluster + ": Applying antrea cloud manifest")
		err = kubeCtl.Apply("", []byte(antreaCloudManifest))
		Expect(err).ToNot(HaveOccurred())

		By(cluster + ": Check antrea cloud is ready")
		err = utils.RestartOrWaitDeployment(cl, "antreacloud-cloud-controller", "antreacloud-system", time.Second*120, false)
		Expect(err).ToNot(HaveOccurred())

		cloudCluster = utils.IsCloudCluster(config.GinkgoConfig.FocusStrings, cloudClusters)
	}
	// Check create VPC status.
	By("Check VM VPCs are ready")
	for {
		err, more := <-wgChan
		if !more {
			for provider, vpc := range cloudVPCs {
				logf.Log.Info("VM VPCs created", "Provider", provider, "VPCID", vpc.GetVPCID())
			}
			break
		}
		Expect(err).ToNot(HaveOccurred())
	}

	if len(k8sClients) == 1 {
		k8sClient = k8sClients[clusters[0]]
	}
	if len(cloudVPCs) == 1 {
		provider := strings.Split(cloudProviders, ",")[0]
		cloudVPC = cloudVPCs[provider]
	}
	close(done)
}, 600)

var _ = AfterSuite(func(done Done) {
	if preserveSetup {
		logf.Log.Info("Preserve setup after tests")
		close(done)
		return
	}
	var controllersCored *string
	var err error
	for _, cluster := range clusters {
		kubeCtl.SetContext(cluster)
		if len(cluster) == 0 {
			cluster = "default"
		}
		By(cluster + ": Check for controllers' cores")
		err = utils.CheckRestart(kubeCtl)
		if err != nil {
			cl := cluster
			controllersCored = &cl
			break
		}
	}
	if controllersCored != nil {
		if preserveSetupOnFail {
			logf.Log.Info("Preserve setup, restart detected")
			close(done)
			return
		}
		if len(supportBundleDir) > 0 {
			logf.Log.Info("Controllers restart detected, collect support bundles", "Cluster", *controllersCored)
			for _, cluster := range clusters {
				utils.CollectSupportBundle(kubeCtl, path.Join(supportBundleDir, cluster, "integration"))
			}
		}
	}
	// Delete VM VPC in parallel.
	wg := sync.WaitGroup{}
	wgChan := make(chan error)
	for provider, v := range cloudVPCs {
		vpc := v
		logf.Log.Info("Initiating deleting VM VPC", "Provider", provider, "VPCID", vpc.GetVPCID())
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := vpc.Delete(time.Second * 600)
			wgChan <- err
		}()
	}
	go func() {
		wg.Wait()
		close(wgChan)
	}()

	// Check delete VPC status.
	By("Waiting for deleting VM VPCs")
	for {
		err, more := <-wgChan
		if !more {
			break
		}
		Expect(err).ToNot(HaveOccurred())
	}
	// Last, consider controller core as failure.
	Expect(controllersCored).To(BeNil(), "Controller cores found")
	close(done)
}, 600)
