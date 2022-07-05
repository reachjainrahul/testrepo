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

package testing

import (
	"fmt"

	antreatypes "antrea.io/antrea/pkg/apis/crd/v1alpha2"
	cloud "antrea.io/antreacloud/apis/crd/v1alpha1"
	"antrea.io/antreacloud/pkg/converter/source"
	"antrea.io/antreacloud/pkg/converter/target"
)

func SetupVirtualMachine(vm *cloud.VirtualMachine, name, namespace string, nics ...*cloud.NetworkInterface) {
	vm.Status.NetworkInterfaces = nil
	for _, nic := range nics {
		ref := cloud.NetworkInterfaceReference{Name: nic.Name, Namespace: nic.Namespace}
		vm.Status.NetworkInterfaces = append(vm.Status.NetworkInterfaces, ref)
	}
	vm.Name = name
	vm.Namespace = namespace
	vm.Status.Tags = map[string]string{"test-vm-tag": "test-vm-key"}
	vm.Status.VirtualPrivateCloud = "test-vm-vpc"
}

func SetupVirtualMachineOwnerOf(vm *source.VirtualMachineSource, name, namespace string,
	nics ...*cloud.NetworkInterface) {
	SetupVirtualMachine(&vm.VirtualMachine, name, namespace, nics...)
}

func SetupNetworkInterface(nic *cloud.NetworkInterface, name, namespace string, ips []string) {
	nic.Name = name
	nic.Namespace = namespace
	nic.Status.Tags = map[string]string{"test-nic-tag": "test-nic-key"}
	nic.Status.IPs = nil
	for _, ip := range ips {
		nic.Status.IPs = append(nic.Status.IPs, cloud.IPAddress{Address: ip})
	}
}

// SetupExternalEntitySources returns externalEntitySource resources for testing.
func SetupExternalEntitySources(ips []string, ports []antreatypes.NamedPort, namespace string) (
	map[string]target.ExternalEntitySource,
	[]*cloud.NetworkInterface,
) {
	sources := make(map[string]target.ExternalEntitySource)
	virtualMachine := &source.VirtualMachineSource{}
	sources["VirtualMachine"] = virtualMachine

	networkInterfaces := make([]*cloud.NetworkInterface, 0)
	for i, ip := range ips {
		name := "nic" + fmt.Sprintf("%d", i)
		nic := &cloud.NetworkInterface{}
		SetupNetworkInterface(nic, name, namespace, []string{ip})
		networkInterfaces = append(networkInterfaces, nic)
	}
	SetupVirtualMachineOwnerOf(virtualMachine, "test-vm", namespace, networkInterfaces...)
	return sources, networkInterfaces
}
