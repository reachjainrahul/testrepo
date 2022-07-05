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

package target_test

import (
	converter "antrea.io/antreacloud/pkg/converter/target"
	"context"
	"fmt"
	"reflect"

	mock "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	antreatypes "antrea.io/antrea/pkg/apis/crd/v1alpha2"

	cloud "antrea.io/antreacloud/apis/crd/v1alpha1"
	"antrea.io/antreacloud/pkg/testing"
	"antrea.io/antreacloud/pkg/testing/controllerruntimeclient"
)

var _ = Describe("Externalentity", func() {

	var (
		// Test framework
		mockCtrl   *mock.Controller
		mockclient *controllerruntimeclient.MockClient

		// Test parameters
		namespace = "test-externalentity-sources-namespace"

		// Test Tunables
		nicGetError                 error
		networkInterfaceIPAddresses = []string{"1.1.1.1", "2.2.2.2"}
		namedports                  = []antreatypes.NamedPort{
			{Name: "http", Protocol: v1.ProtocolTCP, Port: 80},
			{Name: "https", Protocol: v1.ProtocolTCP, Port: 443},
		}

		externalEntitySources map[string]converter.ExternalEntitySource
		networkInterfaces     []*cloud.NetworkInterface
	)

	BeforeEach(func() {
		mockCtrl = mock.NewController(GinkgoT())
		mockclient = controllerruntimeclient.NewMockClient(mockCtrl)
		nicGetError = nil
		externalEntitySources, networkInterfaces = testing.SetupExternalEntitySources(networkInterfaceIPAddresses, namedports, namespace)
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	getEndPointAddressesTester := func(name string, hasNic bool) {
		externalEntitySource := externalEntitySources[name]
		if hasNic {
			orderedCalls := make([]*mock.Call, 0)
			for _, in := range networkInterfaces {
				// Need to explicit return NetworkInterface objects so that its IPs may be retrieved.
				nic := in
				key := client.ObjectKey{Name: nic.Name, Namespace: namespace}
				orderedCalls = append(orderedCalls,
					mockclient.EXPECT().Get(mock.Any(), key, mock.Any()).
						Return(nicGetError).
						Do(func(_ context.Context, key client.ObjectKey, iface *cloud.NetworkInterface) {
							nic.DeepCopyInto(iface)
						}))
			}
			mock.InOrder(orderedCalls...)
		}

		ips, err := externalEntitySource.GetEndPointAddresses(mockclient)
		Expect(err).ToNot(HaveOccurred())
		// As Equal and []string{} == nil
		Expect(ips).To(ConsistOf(networkInterfaceIPAddresses))
		Expect(networkInterfaceIPAddresses).To(ConsistOf(ips))
	}

	getEndPointPortTester := func(name string, hasPort bool) {
		externalEntitySource := externalEntitySources[name]
		ports := externalEntitySource.GetEndPointPort(mockclient)
		if hasPort {
			Expect(ports).To(Equal(namedports))
		} else {
			Expect(ports).To(BeNil())
		}
	}

	copyTester := func(name string) {
		externalEntitySource := externalEntitySources[name]
		dup := externalEntitySource.Copy()
		Expect(dup).To(Equal(externalEntitySource))
	}

	embedTypeTester := func(name string, expType interface{}) {
		externalEntitySource := externalEntitySources[name]
		embed := externalEntitySource.EmbedType()
		Expect(reflect.TypeOf(embed).Elem()).To(Equal(reflect.TypeOf(expType).Elem()))
	}

	getTagsTester := func(name string, hasTag bool) {
		externalEntitySource := externalEntitySources[name]
		tags := externalEntitySource.GetTags()
		if hasTag {
			Expect(tags).ToNot(BeNil())
		} else {
			Expect(tags).To(BeNil())
		}
	}

	getLabelsTester := func(name string, hasLabels bool) {
		mockclient.EXPECT().Get(mock.Any(), mock.Any(), mock.Any()).
			Return(nil).
			Do(func(_ context.Context, key client.ObjectKey, out *cloud.VirtualMachine) {
				vm := externalEntitySources["VirtualMachine"].EmbedType().(*cloud.VirtualMachine)
				Expect(key.Name).To(Equal(vm.Name))
				Expect(key.Namespace).To(Equal(vm.Namespace))
				vm.DeepCopyInto(out)
			}).AnyTimes()

		externalEntitySource := externalEntitySources[name]
		labels := externalEntitySource.GetLabelsFromClient(mockclient)
		if hasLabels {
			Expect(labels).ToNot(BeNil())
		} else {
			Expect(labels).To(BeNil())
		}
	}

	Context("Source has required information", func() {
		table.DescribeTable("GetEndPointAddresses",
			func(name string, hasNic bool) {
				getEndPointAddressesTester(name, hasNic)
			},
			table.Entry("VirtualMachine", "VirtualMachine", true))

		table.DescribeTable("GetEndPointPort",
			func(name string, hasPort bool) {
				getEndPointPortTester(name, hasPort)
			},
			table.Entry("VirtualMachine", "VirtualMachine", false))

		table.DescribeTable("GetTags",
			func(name string, hasTags bool) {
				getTagsTester(name, hasTags)
			},
			table.Entry("VirtualMachine", "VirtualMachine", true))

		table.DescribeTable("GetLabels",
			func(name string, hasLabels bool) {
				getLabelsTester(name, hasLabels)
			},
			table.Entry("VirtualMachine", "VirtualMachine", true))

		table.DescribeTable("Copy",
			func(name string) {
				copyTester(name)
			},
			table.Entry("VirtualMachine", "VirtualMachine"))

		table.DescribeTable("EmbedType",
			func(name string, expType interface{}) {
				embedTypeTester(name, expType)
			},
			table.Entry("VirtualMachine", "VirtualMachine", &cloud.VirtualMachine{}))
	})

	Context("Source has not required information", func() {
		JustBeforeEach(func() {
			networkInterfaceIPAddresses = nil
			nicGetError = fmt.Errorf("dummy")
			namedports = nil
		})

		table.DescribeTable("GetEndPointAddresses",
			func(name string, hasNic bool) {
				getEndPointAddressesTester(name, hasNic)
			},
			table.Entry("VirtualMachine", "VirtualMachine", true))

		table.DescribeTable("GetEndPointPort",
			func(name string, hasPort bool) {
				getEndPointPortTester(name, hasPort)
			},
			table.Entry("VirtualMachine", "VirtualMachine", false))
	})

})
