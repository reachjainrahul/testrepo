/*******************************************************************************
 * Copyright 2022 VMWare, Inc.  All rights reserved. -- VMWare Confidential
 *******************************************************************************/

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

// NetworkInterface is the Schema for the networkinterfaces API
type NetworkInterface struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetworkInterfaceSpec   `json:"spec,omitempty"`
	Status NetworkInterfaceStatus `json:"status,omitempty"`
}

// NetworkInterfaceList contains a list of NetworkInterface
type NetworkInterfaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NetworkInterface `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NetworkInterface{}, &NetworkInterfaceList{})
}
