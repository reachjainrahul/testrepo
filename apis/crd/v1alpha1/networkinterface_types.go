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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AddressType specifies IP address type.
type AddressType string

const (
	// Address type is host name.
	AddressTypeHostName AddressType = "HostName"
	// Address type is internal IP.
	AddressTypeInternalIP AddressType = "InternalIP"
	// Address type is external IP.
	AddressTypeExternalIP AddressType = "ExternalIP"
)

type IPAddress struct {
	AddressType AddressType `json:"addressType"`
	Address     string      `json:"address"`
}

// NetworkInterfaceReference references to NetworkInterface.
type NetworkInterfaceReference struct {
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

// NetworkInterfaceSpec defines the desired state of NetworkInterface
// It contains user tunable parameters.
type NetworkInterfaceSpec struct {
}

// NetworkInterfaceStatus contains information pertaining to NetworkInterface.
type NetworkInterfaceStatus struct {
	// Tags of this interface. A corresponding label is also generated for each tag.
	Tags map[string]string `json:"tags,omitempty"`
	// Hardware address of the interface.
	MAC string `json:"mac,omitempty"`
	// IP addresses of this NetworkInterface.
	IPs []IPAddress `json:"ips,omitempty"`
	// NetworkPolicies indicates NetworkPolicy status on this NetworkInterface.
	NetworkPolicies map[string]string `json:"networkPolicies,omitempty"`
}

// +kubebuilder:object:root=true

// +kubebuilder:resource:shortName="ni"
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Owner-Id",type=string,JSONPath=`.metadata.ownerReferences[0].name`
// +kubebuilder:printcolumn:name="Owner-Type",type=string,JSONPath=`.metadata.ownerReferences[0].kind`
// +kubebuilder:printcolumn:name="Internal-IP",type=string,JSONPath=`.status.ips[?(@.addressType == 'InternalIP')].address`
// +kubebuilder:printcolumn:name="External-IP",type=string,JSONPath=`.status.ips[?(@.addressType == 'ExternalIP')].address`
// NetworkInterface is the Schema for the networkinterfaces API.
// A NetworkInterface is generated as part of VirtualMachine
// resource creation.
type NetworkInterface struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetworkInterfaceSpec   `json:"spec,omitempty"`
	Status NetworkInterfaceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NetworkInterfaceList contains a list of NetworkInterface.
type NetworkInterfaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NetworkInterface `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NetworkInterface{}, &NetworkInterfaceList{})
}
