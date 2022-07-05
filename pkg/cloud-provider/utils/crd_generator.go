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
	"fmt"
	"strings"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"

	cloudv1alpha1 "antrea.io/antreacloud/apis/crd/v1alpha1"
	cloudcommon "antrea.io/antreacloud/pkg/cloud-provider/cloudapi/common"
)

func GenerateVirtualMachineCRD(crdName string, cloudName string, cloudID string, namespace string, cloudNetwork string,
	shortNetworkID string, status string, tags map[string]string, provider cloudcommon.ProviderType) *cloudv1alpha1.VirtualMachine {
	vmStatus := &cloudv1alpha1.VirtualMachineStatus{
		Provider:            cloudv1alpha1.CloudProvider(provider),
		VirtualPrivateCloud: shortNetworkID,
		Tags:                tags,
		Status:              status,
	}
	annotationsMap := map[string]string{
		cloudcommon.AnnotationCloudAssignedIDKey:    cloudID,
		cloudcommon.AnnotationCloudAssignedNameKey:  cloudName,
		cloudcommon.AnnotationCloudAssignedVPCIDKey: cloudNetwork,
	}

	vmCrd := &cloudv1alpha1.VirtualMachine{
		TypeMeta: v1.TypeMeta{
			Kind:       cloudcommon.VirtualMachineCRDKind,
			APIVersion: cloudcommon.APIVersion,
		},
		ObjectMeta: v1.ObjectMeta{
			UID:         uuid.NewUUID(),
			Name:        crdName,
			Namespace:   namespace,
			Annotations: annotationsMap,
		},
		Spec:   cloudv1alpha1.VirtualMachineSpec{},
		Status: *vmStatus,
	}

	return vmCrd
}

func UpdateVirtualMachineCRDWithNetworkInterfaceRef(virtualMachine *cloudv1alpha1.VirtualMachine,
	networkInterfaces []*cloudv1alpha1.NetworkInterface) {
	for _, networkInterface := range networkInterfaces {
		networkInterfaceReference := cloudv1alpha1.NetworkInterfaceReference{
			Name:      networkInterface.GetName(),
			Namespace: networkInterface.GetNamespace(),
		}
		virtualMachine.Status.NetworkInterfaces = append(virtualMachine.Status.NetworkInterfaces, networkInterfaceReference)
	}
}

func GenerateNetworkInterfaceCRD(crdName string, cloudName string, cloudID string, namespace string, network string,
	privateIPs []string, publicIPs []string, macAddr string, tags map[string]string, ownerName string, ownerUID types.UID,
	ownerKind string) *cloudv1alpha1.NetworkInterface {
	var ipAddressCRDs []cloudv1alpha1.IPAddress

	if len(privateIPs) > 0 {
		for _, ipAddress := range privateIPs {
			ipAddressCRD := cloudv1alpha1.IPAddress{
				AddressType: cloudv1alpha1.AddressTypeInternalIP,
				Address:     ipAddress,
			}
			ipAddressCRDs = append(ipAddressCRDs, ipAddressCRD)

			if len(publicIPs) > 0 {
				for _, publicIP := range publicIPs {
					ipAddressCRD := cloudv1alpha1.IPAddress{
						AddressType: cloudv1alpha1.AddressTypeExternalIP,
						Address:     publicIP,
					}
					ipAddressCRDs = append(ipAddressCRDs, ipAddressCRD)
				}
			}
		}
	}

	var trueVar = true
	ownerRef := v1.OwnerReference{
		APIVersion:         cloudcommon.APIVersion,
		Kind:               ownerKind,
		Name:               ownerName,
		UID:                ownerUID,
		Controller:         nil,
		BlockOwnerDeletion: &trueVar,
	}
	annotationsMap := map[string]string{
		cloudcommon.AnnotationCloudAssignedIDKey:    strings.ToLower(cloudID),
		cloudcommon.AnnotationCloudAssignedNameKey:  strings.ToLower(cloudName),
		cloudcommon.AnnotationCloudAssignedVPCIDKey: strings.ToLower(network),
	}

	networkInterfaceCRD := &cloudv1alpha1.NetworkInterface{
		TypeMeta: v1.TypeMeta{
			Kind:       cloudcommon.NetworkInterfaceCRDKind,
			APIVersion: cloudcommon.APIVersion,
		},
		ObjectMeta: v1.ObjectMeta{
			Name:            crdName,
			Namespace:       namespace,
			UID:             uuid.NewUUID(),
			OwnerReferences: []v1.OwnerReference{ownerRef},
			Annotations:     annotationsMap,
		},
		Spec: cloudv1alpha1.NetworkInterfaceSpec{},
		Status: cloudv1alpha1.NetworkInterfaceStatus{
			Tags: tags,
			MAC:  macAddr,
			IPs:  ipAddressCRDs,
		},
	}

	return networkInterfaceCRD
}

func GenerateShortResourceIdentifier(id string, prefixToAdd string) string {
	idTrim := strings.Trim(id, " ")
	if len(idTrim) == 0 {
		return ""
	}

	// Ascii value of the characters will be added to generate unique name
	var sum uint32 = 0
	for _, value := range strings.ToLower(idTrim) {
		sum += uint32(value)
	}

	str := fmt.Sprintf("%v-%v", strings.ToLower(prefixToAdd), sum)
	return str
}
