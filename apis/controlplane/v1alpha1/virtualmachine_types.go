/*******************************************************************************
 * Copyright 2022 VMWare, Inc.  All rights reserved. -- VMWare Confidential
 *******************************************************************************/

package v1alpha1

import (
	crd "antrea.io/antreacloud/apis/crd/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VirtualMachineSpec defines the desired state of VirtualMachine.
type VirtualMachineSpec struct {
}

// VirtualMachineStatus defines the observed state of VirtualMachine
// It contains observable parameters.
type VirtualMachineStatus struct {
	// Provider specifies cloud provider of this VirtualMachine.
	Provider crd.CloudProvider `json:"provider,omitempty"`
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

// VirtualMachine is the Schema for the virtualmachines API
type VirtualMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualMachineSpec   `json:"spec,omitempty"`
	Status VirtualMachineStatus `json:"status,omitempty"`
}

// VirtualMachineList contains a list of VirtualMachine
type VirtualMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualMachine `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VirtualMachine{}, &VirtualMachineList{})
}
