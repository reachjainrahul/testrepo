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

package aws

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws/endpoints"

	"antrea.io/antreacloud/apis/crd/v1alpha1"
	"antrea.io/antreacloud/pkg/cloud-provider/cloudapi/internal"
)

type awsAccountCredentials struct {
	accountID       string
	accessKeyID     string
	accessKeySecret string
	region          string
	roleArn         string
	externalID      string
}

// validateAccountCredentials validates account config.
func validateAccountCredentials(credentials interface{}) (interface{}, error) {
	configAWS := credentials.(*v1alpha1.CloudProviderAccountConfigAWS)
	accCreds := &awsAccountCredentials{
		accountID:       strings.TrimSpace(configAWS.AccountID),
		accessKeyID:     strings.TrimSpace(configAWS.AccessKeyID),
		accessKeySecret: strings.TrimSpace(configAWS.AccessKeySecret),
		region:          strings.TrimSpace(configAWS.Region),
		roleArn:         strings.TrimSpace(configAWS.RoleArn),
		externalID:      strings.TrimSpace(configAWS.ExternalID),
	}

	// validate account ID
	if len(accCreds.accountID) == 0 {
		return nil, fmt.Errorf("account id cannot be blank or empty")
	}

	// warning for using role based auth
	if len(accCreds.roleArn) != 0 {
		awsPluginLogger().Info("Role ARN configured will be used for cloud-account access")
		// empty credentials when role based access is configured
		accCreds.accessKeyID = ""
		accCreds.accessKeySecret = ""
	} else if len(accCreds.accessKeyID) == 0 || len(accCreds.accessKeySecret) == 0 {
		return nil, fmt.Errorf("must specify either credentials or role arn, cannot both be empty")
	}

	// validate region
	if len(accCreds.region) == 0 {
		return nil, fmt.Errorf("region cannot be blank or empty")
	}
	// NOTE: currently only AWS standard partition regions supported (aws-cn, aws-us-gov etc are not
	// supported). As we add support for other partitions, validation needs to be updated
	regions := endpoints.AwsPartition().Regions()
	_, found := regions[accCreds.region]
	if !found {
		var supportedRegions []string
		for key := range regions {
			supportedRegions = append(supportedRegions, key)
		}
		return nil, fmt.Errorf("%v not in supported regions [%v]", accCreds.region, supportedRegions)
	}

	return accCreds, nil
}

func compareAccountCredentials(accountName string, existing interface{}, new interface{}) bool {
	existingCreds := existing.(*awsAccountCredentials)
	newCreds := new.(*awsAccountCredentials)

	credsChanged := false
	if strings.Compare(existingCreds.accountID, newCreds.accountID) != 0 {
		credsChanged = true
		awsPluginLogger().Info("account ID updated", "account", accountName)
	}
	if strings.Compare(existingCreds.accessKeyID, newCreds.accessKeyID) != 0 {
		credsChanged = true
		awsPluginLogger().Info("account access key ID updated", "account", accountName)
	}
	if strings.Compare(existingCreds.accessKeySecret, newCreds.accessKeySecret) != 0 {
		credsChanged = true
		awsPluginLogger().Info("account access key secret updated", "account", accountName)
	}
	if strings.Compare(existingCreds.region, newCreds.region) != 0 {
		credsChanged = true
		awsPluginLogger().Info("account region updated", "account", accountName)
	}
	return credsChanged
}

// getVpcAccount returns first found account config to which this vpc id belongs.
func (c *awsCloud) getVpcAccount(vpcID string) internal.CloudAccountInterface {
	accCfgs := c.cloudCommon.GetCloudAccounts()
	if len(accCfgs) == 0 {
		return nil
	}

	for _, accCfg := range accCfgs {
		ec2ServiceCfg, err := accCfg.GetServiceConfigByName(awsComputeServiceNameEC2)
		if err != nil {
			awsPluginLogger().Error(err, "get ec2 service config failed", "vpcID", vpcID, "account", accCfg.GetNamespacedName())
			continue
		}
		accVpcIDs := ec2ServiceCfg.(*ec2ServiceConfig).getCachedVpcIDs()
		if len(accVpcIDs) == 0 {
			awsPluginLogger().Info("no vpc found for account", "vpcID", vpcID, "account", accCfg.GetNamespacedName())
			continue
		}
		if _, found := accVpcIDs[strings.ToLower(vpcID)]; found {
			return accCfg
		}
		awsPluginLogger().Info("vpcID not found in cache", "vpcID", vpcID, "account", accCfg.GetNamespacedName())
	}
	return nil
}
