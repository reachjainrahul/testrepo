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

// EntityMatch specifies match conditions to cloud entities.
// Cloud entities must satisfy all fields(ANDed) in EntityMatch to satisfy EntityMatch.
type EntityMatch struct {
	// MatchName matches cloud entities' name. If not specified, it matches any cloud entities.
	MatchName string `json:"matchName,omitempty"`
	// MatchID matches cloud entities' identifier. If not specified, it matches any cloud entities.
	MatchID string `json:"matchID,omitempty"`
	// MatchName matches cloud entities's tags. Tag matches are ANDed.
	// If not specified, it matches any cloud entities.
	MatchTags map[string]string `json:"matchTags,omitempty"`
}

// VirtualMachineMatch specifies VirtualMachine match criteria.
// VirtualMachines must satisfy all fields(ANDed) in VirtualMachineMatch in order to satisfy VirtualMachineMatch.
type VirtualMachineMatch struct {
	// VpcMatch specifies the virtual private cloud to which VirtualMachines belong.
	// If it is not specified, VirtualMachines may belong to any virtual private cloud,
	VpcMatch *EntityMatch `json:"vpcMatch,omitempty"`
	// VMMatch specifies VirtualMachines to match.
	// If it is not specified, all VirtualMachines are matching.
	VMMatch *EntityMatch `json:"vmMatch,omitempty"`
}

// VirtualMachineSelector specifies VM selection criteria.
type VirtualMachineSelector struct {
	// VMMatches is an array of VirtualMachineMatch.
	// VirtualMachines satisfying any item on VMMatches are selected(ORed).
	// If not specified, all VirtualMachines are selected.
	VMMatches []VirtualMachineMatch `json:"vmMatches,omitempty"`
}

// CloudEntitySelectorSpec defines the desired state of CloudEntitySelector.
type CloudEntitySelectorSpec struct {
	// AccountName specifies cloud account in this CloudProvider.
	AccountName string `json:"accountName,omitempty"`
	// VMSelector selects the VirtualMachines the user has modify privilege.
	// If it is not specified, no VirtualMachines are selected.
	VMSelector *VirtualMachineSelector `json:"vmSelector,omitempty"`
}

// CloudEntitySelectorStatus defines the current state of CloudEntitySelector.
type CloudEntitySelectorStatus struct {
	// Status is current state of CloudEntitySelector.
	Status string `json:"Status,omitempty"`
}

// +kubebuilder:object:root=true

// +kubebuilder:resource:shortName="ces"
// CloudEntitySelector is the Schema for the cloudentityselectors API.
type CloudEntitySelector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              CloudEntitySelectorSpec   `json:"spec,omitempty"`
	Status            CloudEntitySelectorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CloudEntitySelectorList contains a list of CloudEntitySelector.
type CloudEntitySelectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CloudEntitySelector `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CloudEntitySelector{}, &CloudEntitySelectorList{})
}
