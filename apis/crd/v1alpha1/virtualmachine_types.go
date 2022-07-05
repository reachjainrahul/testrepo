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

// VirtualMachineSpec defines the desired state of VirtualMachine.
type VirtualMachineSpec struct {
}

// VirtualMachineStatus defines the observed state of VirtualMachine
// It contains observable parameters.
type VirtualMachineStatus struct {
	// Provider specifies cloud provider of this VirtualMachine.
	Provider CloudProvider `json:"provider,omitempty"`
	// VirtualPrivateCloud is the virtual private cloud this VirtualMachine belongs to.
	VirtualPrivateCloud string `json:"virtualPrivateCloud,omitempty"`
	// Tags of this VirtualMachine. A corresponding label is also generated for each tag.
	Tags map[string]string `json:"tags,omitempty"`
	// NetworkInterfaces is array of NetworkInterfaces attached to this VirtualMachine.
	NetworkInterfaces []NetworkInterfaceReference `json:"networkInterfaces,omitempty"`
	// Status indicates current state of the VirtualMachine.
	Status string `json:"status,omitempty"`
	// NetworkPolicies indicates NetworkPolicy status on this VirtualMachine.
	NetworkPolicies map[string]string `json:"networkPolicies,omitempty"`
	// Error is current error, if any, of the VirtualMachine.
	Error string `json:"error,omitempty"`
}

// +genclient
// +kubebuilder:object:root=true

// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName="vm"
// +kubebuilder:printcolumn:name="Cloud-Provider",type=string,JSONPath=`.status.provider`
// +kubebuilder:printcolumn:name="Virtual-Private-Cloud",type=string,JSONPath=`.status.virtualPrivateCloud`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.status`
// VirtualMachine is the Schema for the virtualmachines API
// A virtualMachine object is created automatically based on
// matching criteria specification of CloudEntitySelector.
type VirtualMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualMachineSpec   `json:"spec,omitempty"`
	Status VirtualMachineStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// VirtualMachineList contains a list of VirtualMachine.
type VirtualMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualMachine `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VirtualMachine{}, &VirtualMachineList{})
}
