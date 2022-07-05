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

package aws

import (
	"github.com/aws/aws-sdk-go/service/ec2"

	"antrea.io/antreacloud/apis/crd/v1alpha1"
	cloudcommon "antrea.io/antreacloud/pkg/cloud-provider/cloudapi/common"
	"antrea.io/antreacloud/pkg/cloud-provider/utils"
)

const ResourceNameTagKey = "Name"

// ec2InstanceToVirtualMachineCRD converts ec2 instance to VirtualMachine CRD.
func ec2InstanceToVirtualMachineCRD(instance *ec2.Instance, namespace string) *v1alpha1.VirtualMachine {
	tags := make(map[string]string)
	vmTags := instance.Tags
	if len(vmTags) > 0 {
		for _, tag := range vmTags {
			tags[*tag.Key] = *tag.Value
		}
	}

	cloudName := tags[ResourceNameTagKey]
	cloudID := *instance.InstanceId
	cloudNetwork := *instance.VpcId

	return utils.GenerateVirtualMachineCRD(cloudID, cloudName, cloudID, namespace, cloudNetwork, cloudNetwork,
		*instance.State.Name, tags, providerType)
}

//  ec2InstanceToNetworkInterfaceCRD converts ec2 instance to NetworkInterface CRDs with VM as owner.
func ec2InstanceToNetworkInterfaceCRD(networkInterfaces []*ec2.InstanceNetworkInterface, owner *v1alpha1.VirtualMachine,
	namespace string) []*v1alpha1.NetworkInterface {
	networkInterfaceCRDs := make([]*v1alpha1.NetworkInterface, 0, len(networkInterfaces))
	for _, nwInf := range networkInterfaces {
		var privateIPs []string
		var publicIPs []string

		privateIPAddresses := nwInf.PrivateIpAddresses
		if len(privateIPAddresses) > 0 {
			for _, ipAddress := range privateIPAddresses {
				privateIPs = append(privateIPs, *ipAddress.PrivateIpAddress)
				association := ipAddress.Association
				if association != nil {
					publicIPs = append(publicIPs, *association.PublicIp)
				}
			}
		}

		cloudID := *nwInf.NetworkInterfaceId
		macAddress := *nwInf.MacAddress
		cloudNetwork := *nwInf.VpcId

		networkInterfaceCRD := utils.GenerateNetworkInterfaceCRD(cloudID, "", cloudID, namespace, cloudNetwork,
			privateIPs, publicIPs, macAddress, nil, owner.Name, owner.UID, cloudcommon.VirtualMachineCRDKind)
		networkInterfaceCRDs = append(networkInterfaceCRDs, networkInterfaceCRD)
	}

	return networkInterfaceCRDs
}
