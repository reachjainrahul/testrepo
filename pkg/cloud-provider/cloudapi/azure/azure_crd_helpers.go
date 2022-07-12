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

package azure

import (
	"strings"

	"antrea.io/antreacloud/apis/crd/v1alpha1"
	"antrea.io/antreacloud/pkg/cloud-provider/securitygroup"
	"antrea.io/antreacloud/pkg/cloud-provider/utils"
)

func computeInstanceToVirtualMachineCRD(instance *virtualMachineTable, namespace string) *v1alpha1.VirtualMachine {
	tags := make(map[string]string)
	vmTags := instance.Tags
	for key, value := range vmTags {
		// skip any tags added by antreacloud for internal processing
		_, hasAGPrefix, hasATPrefix := securitygroup.IsAntreaCloudCreatedSecurityGroup(key)
		if hasAGPrefix || hasATPrefix {
			continue
		}
		tags[key] = *value
	}

	//Network interface associated with Virtual machine
	instNetworkInterfaces := instance.NetworkInterfaces
	networkInterfaces := make([]v1alpha1.NetworkInterface, 0, len(instNetworkInterfaces))
	for _, nwInf := range instNetworkInterfaces {
		var ipAddressCRDs []v1alpha1.IPAddress
		if len(nwInf.PrivateIps) > 0 {
			for _, ipAddress := range nwInf.PrivateIps {
				ipAddressCRD := v1alpha1.IPAddress{
					AddressType: v1alpha1.AddressTypeInternalIP,
					Address:     *ipAddress,
				}
				ipAddressCRDs = append(ipAddressCRDs, ipAddressCRD)
			}
		}
		if len(nwInf.PublicIps) > 0 {
			for _, publicIP := range nwInf.PublicIps {
				ipAddressCRD := v1alpha1.IPAddress{
					AddressType: v1alpha1.AddressTypeInternalIP,
					Address:     *publicIP,
				}
				ipAddressCRDs = append(ipAddressCRDs, ipAddressCRD)
			}
		}
		macAddress := ""
		if nwInf.MacAddress != nil {
			macAddress = *nwInf.MacAddress
		}

		nw_tags := make(map[string]string)
		for key, value := range nwInf.Tags {
			// skip any tags added by antreacloud for internal processing
			_, hasAGPrefix, hasATPrefix := securitygroup.IsAntreaCloudCreatedSecurityGroup(key)
			if hasAGPrefix || hasATPrefix {
				continue
			}
			nw_tags[key] = *value
		}
		networkInterface := v1alpha1.NetworkInterface{
			Name: *nwInf.ID,
			Tags: nw_tags,
			MAC:  macAddress,
			IPs:  ipAddressCRDs,
		}
		networkInterfaces = append(networkInterfaces, networkInterface)
	}

	cloudNetworkID := strings.ToLower(*instance.VnetID)
	cloudID := strings.ToLower(*instance.ID)
	cloudName := strings.ToLower(*instance.Name)
	crdName := utils.GenerateShortResourceIdentifier(cloudID, cloudName)

	_, _, nwResName, err := extractFieldsFromAzureResourceID(cloudNetworkID)
	if err != nil {
		azurePluginLogger().Error(err, "failed to create VirtualMachine CRD")
		return nil
	}
	cloudNetworkShortID := utils.GenerateShortResourceIdentifier(cloudNetworkID, nwResName)
	return utils.GenerateVirtualMachineCRD(crdName, strings.ToLower(cloudName), strings.ToLower(cloudID), namespace,
		strings.ToLower(cloudNetworkID), cloudNetworkShortID,
		*instance.Status, tags, networkInterfaces, providerType)
}
