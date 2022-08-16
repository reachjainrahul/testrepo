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
	"context"
	"encoding/json"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"

	"antrea.io/nephe/apis/crd/v1alpha1"
	"antrea.io/nephe/pkg/cloud-provider/cloudapi/internal"
)

type azureAccountCredentials struct {
	SubscriptionID string `json:"subscriptionId,omitempty"`
	ClientID       string `json:"clientId,omitempty"`
	TenantID       string `json:"tenantId,omitempty"`
	ClientKey      string `json:"clientKey,omitempty"`
	region         string
}

// setAccountCredentials sets account credentials.
func setAccountCredentials(client client.Client, credentials interface{}) (interface{}, error) {
	azureConfig := credentials.(*v1alpha1.CloudProviderAccountAzureConfig)
	accCreds, err := extractSecret(client, azureConfig.SecretRef)
	if err != nil {
		return nil, err
	}
	accCreds.region = strings.TrimSpace(azureConfig.Region)

	return accCreds, nil
}

func compareAccountCredentials(accountName string, existing interface{}, new interface{}) bool {
	existingCreds := existing.(*azureAccountCredentials)
	newCreds := new.(*azureAccountCredentials)

	credsChanged := false
	if strings.Compare(existingCreds.SubscriptionID, newCreds.SubscriptionID) != 0 {
		credsChanged = true
		azurePluginLogger().Info("subscription ID updated", "account", accountName)
	}
	if strings.Compare(existingCreds.ClientID, newCreds.ClientID) != 0 {
		credsChanged = true
		azurePluginLogger().Info("client ID updated", "account", accountName)
	}
	if strings.Compare(existingCreds.TenantID, newCreds.TenantID) != 0 {
		credsChanged = true
		azurePluginLogger().Info("account tenant ID updated", "account", accountName)
	}
	if strings.Compare(existingCreds.ClientKey, newCreds.ClientKey) != 0 {
		credsChanged = true
		azurePluginLogger().Info("account client key updated", "account", accountName)
	}
	if strings.Compare(existingCreds.region, newCreds.region) != 0 {
		credsChanged = true
		azurePluginLogger().Info("account region updated", "account", accountName)
	}
	return credsChanged
}

// ExtractSecret extracts credentials from a Kubernetes secret.
func extractSecret(client client.Client, s *v1alpha1.SecretReference) (*azureAccountCredentials, error) {
	if s == nil {
		return nil, fmt.Errorf("secret reference not found")
	}
	secret := &corev1.Secret{}
	if err := client.Get(context.TODO(), types.NamespacedName{Namespace: s.Namespace, Name: s.Name}, secret); err != nil {
		return nil, fmt.Errorf("unable to get secret: %s", err.Error())
	}
	cred := &azureAccountCredentials{}
	if err := json.Unmarshal(secret.Data[s.Key], cred); err != nil {
		return nil, fmt.Errorf("unable to parse secret data: %s", err.Error())
	}
	return cred, nil
}

// getVnetAccount returns first found account config to which this vnet id belongs.
func (c *azureCloud) getVnetAccount(vpcID string) internal.CloudAccountInterface {
	accCfgs := c.cloudCommon.GetCloudAccounts()
	if len(accCfgs) == 0 {
		return nil
	}

	for _, accCfg := range accCfgs {
		ec2ServiceCfg, err := accCfg.GetServiceConfigByName(azureComputeServiceNameCompute)
		if err != nil {
			continue
		}
		accVpcIDs := ec2ServiceCfg.(*computeServiceConfig).getCachedVnetIDs()
		if len(accVpcIDs) == 0 {
			continue
		}
		if _, found := accVpcIDs[strings.ToLower(vpcID)]; found {
			return accCfg
		}
	}
	return nil
}
