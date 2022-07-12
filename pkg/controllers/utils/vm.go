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

package utils

import (
	"antrea.io/antreacloud/apis/crd/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetVMIPAddresses returns IP addresses of VMs. It returns most appropriate IP is single is true;
// otherwise it turns all IPs.
func GetVMIPAddresses(vm *v1alpha1.VirtualMachine, cl client.Client) []v1alpha1.IPAddress {
	ipLen := len(vm.Status.NetworkInterfaces)
	if ipLen == 0 {
		return nil
	}
	ips := make([]v1alpha1.IPAddress, 0, ipLen)
	for _, value := range vm.Status.NetworkInterfaces {
		ips = append(ips, value.IPs...)
	}
	return ips
}
