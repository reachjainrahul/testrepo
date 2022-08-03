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
	"fmt"
	"strings"

	"antrea.io/cloudcontroller/apis/crd/v1alpha1"
	"antrea.io/cloudcontroller/pkg/cloud-provider/cloudapi/internal"
)

type azureAccountCredentials struct {
	subscriptionID   string
	clientID         string
	tenantID         string
	clientKey        string
	region           string
	identityClientID string
}

// validateAccountCredentials validates account config.
func validateAccountCredentials(credentials interface{}) (interface{}, error) {
	configAzure := credentials.(*v1alpha1.CloudProviderAccountConfigAzure)
	accCreds := &azureAccountCredentials{
		subscriptionID:   strings.TrimSpace(configAzure.SubscriptionID),
		clientID:         strings.TrimSpace(configAzure.ClientID),
		tenantID:         strings.TrimSpace(configAzure.TenantID),
		clientKey:        strings.TrimSpace(configAzure.ClientKey),
		region:           strings.TrimSpace(configAzure.Region),
		identityClientID: strings.TrimSpace(configAzure.IdentityClientID),
	}

	// validate subscription ID
	if len(accCreds.subscriptionID) == 0 {
		return nil, fmt.Errorf("subscription id cannot be blank or empty")
	}

	// validate tenant ID
	if len(accCreds.tenantID) == 0 {
		return nil, fmt.Errorf("tenant id cannot be blank or empty")
	}

	// validate credentials
	if len(accCreds.identityClientID) != 0 {
		azurePluginLogger().Info("Managed Identity Client ID configured will be used for cloud-account access")
		// empty credentials when role based access is configured
		accCreds.clientID = ""
		accCreds.clientKey = ""
	} else if len(accCreds.clientID) == 0 || len(accCreds.clientKey) == 0 {
		return nil, fmt.Errorf("must specify either credentials or managed identity client id, cannot both be empty")
	}

	// validate region
	if len(accCreds.region) == 0 {
		return nil, fmt.Errorf("region cannot be blank or empty")
	}
	return accCreds, nil
}

func compareAccountCredentials(accountName string, existing interface{}, new interface{}) bool {
	existingCreds := existing.(*azureAccountCredentials)
	newCreds := new.(*azureAccountCredentials)

	credsChanged := false
	if strings.Compare(existingCreds.subscriptionID, newCreds.subscriptionID) != 0 {
		credsChanged = true
		azurePluginLogger().Info("subscription ID updated", "account", accountName)
	}
	if strings.Compare(existingCreds.clientID, newCreds.clientID) != 0 {
		credsChanged = true
		azurePluginLogger().Info("client ID updated", "account", accountName)
	}
	if strings.Compare(existingCreds.tenantID, newCreds.tenantID) != 0 {
		credsChanged = true
		azurePluginLogger().Info("account tenant ID updated", "account", accountName)
	}
	if strings.Compare(existingCreds.clientKey, newCreds.clientKey) != 0 {
		credsChanged = true
		azurePluginLogger().Info("account client key updated", "account", accountName)
	}
	if strings.Compare(existingCreds.region, newCreds.region) != 0 {
		credsChanged = true
		azurePluginLogger().Info("account region updated", "account", accountName)
	}
	return credsChanged
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
