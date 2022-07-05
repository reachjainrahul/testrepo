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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// +kubebuilder:validation:Enum=Azure;AWS
// CloudProvider specifies a cloud provider.
type CloudProvider string

const (
	// AzureCloudProvider specifies Azure.
	AzureCloudProvider CloudProvider = "Azure"
	// AWSCloudProvider specifies AWS.
	AWSCloudProvider    CloudProvider = "AWS"
	OnPremCloudProvider CloudProvider = "" // Not a real cloud provider
)

// CloudProviderAccountSpec defines the desired state of CloudProviderAccount.
type CloudProviderAccountSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// PollIntervalInSeconds defines account poll interval (default value is 60, if not specified)
	PollIntervalInSeconds *uint `json:"pollIntervalInSeconds,omitempty"`
	// Cloud provider type
	ProviderType CloudProvider `json:"providerType,omitempty"`
	// Cloud provider account config
	ConfigAWS *CloudProviderAccountConfigAWS `json:"configAWS,omitempty"`
	// Cloud provider account config
	ConfigAzure *CloudProviderAccountConfigAzure `json:"configAzure,omitempty"`
}

type CloudProviderAccountConfigAWS struct {
	// Cloud provider account identifier
	AccountID string `json:"accountID,omitempty"`
	// Cloud provider account access key ID
	AccessKeyID string `json:"accessKeyId,omitempty"`
	// Cloud provider account access key secret
	// (TODO Secret needs to be saved using k8 secrets)
	AccessKeySecret string `json:"accessKeySecret,omitempty"`
	// Cloud provider account region
	Region string `json:"region,omitempty"`
	// Cloud provider role arn to be assumed
	RoleArn string `json:"roleArn,omitempty"`
	// Cloud provider external id used in assume role
	ExternalID string `json:"externalId,omitempty"`
}

type CloudProviderAccountConfigAzure struct {
	SubscriptionID   string `json:"subscriptionId,omitempty"`
	ClientID         string `json:"clientId,omitempty"`
	TenantID         string `json:"tenantId,omitempty"`
	ClientKey        string `json:"clientKey,omitempty"`
	Region           string `json:"region,omitempty"`
	IdentityClientID string `json:"identityClientId,omitempty"`
}

// CloudProviderAccountStatus defines the observed state of CloudProviderAccount.
type CloudProviderAccountStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// Error is current error, if any, of the CloudProviderAccount.
	Error string `json:"error,omitempty"`
}

// +kubebuilder:object:root=true

// +kubebuilder:resource:shortName="cpa"
// +kubebuilder:subresource:status
// CloudProviderAccount is the Schema for the cloudprovideraccounts API.
type CloudProviderAccount struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CloudProviderAccountSpec   `json:"spec,omitempty"`
	Status CloudProviderAccountStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CloudProviderAccountList contains a list of CloudProviderAccount.
type CloudProviderAccountList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CloudProviderAccount `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CloudProviderAccount{}, &CloudProviderAccountList{})
}
