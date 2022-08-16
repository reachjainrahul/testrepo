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
	"context"
	"encoding/json"
	"fmt"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"

	"github.com/aws/aws-sdk-go/aws/endpoints"
	corev1 "k8s.io/api/core/v1"

	"antrea.io/nephe/apis/crd/v1alpha1"
	"antrea.io/nephe/pkg/cloud-provider/cloudapi/internal"
)

type awsAccountCredentials struct {
	accountID       string
	AccessKeyID     string `json:"accessKeyId,omitempty"`
	AccessKeySecret string `json:"accessKeySecret,omitempty"`
	RoleArn         string `json:"roleArn,omitempty"`
	ExternalID      string `json:"externalId,omitempty"`
	region          string
}

// setAccountCredentials sets account credentials.
func setAccountCredentials(client client.Client, credentials interface{}) (interface{}, error) {
	awsConfig := credentials.(*v1alpha1.CloudProviderAccountAWSConfig)
	accCreds, err := extractSecret(client, awsConfig.SecretRef)
	if err != nil {
		return nil, err
	}
	accCreds.accountID = strings.TrimSpace(awsConfig.AccountID)
	accCreds.region = strings.TrimSpace(awsConfig.Region)

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
	if strings.Compare(existingCreds.AccessKeyID, newCreds.AccessKeyID) != 0 {
		credsChanged = true
		awsPluginLogger().Info("account access key ID updated", "account", accountName)
	}
	if strings.Compare(existingCreds.AccessKeySecret, newCreds.AccessKeySecret) != 0 {
		credsChanged = true
		awsPluginLogger().Info("account access key secret updated", "account", accountName)
	}
	if strings.Compare(existingCreds.region, newCreds.region) != 0 {
		credsChanged = true
		awsPluginLogger().Info("account region updated", "account", accountName)
	}
	return credsChanged
}

// ExtractSecret extracts credentials from a Kubernetes secret.
func extractSecret(client client.Client, s *v1alpha1.SecretReference) (*awsAccountCredentials, error) {
	if s == nil {
		return nil, fmt.Errorf("secret reference not found")
	}
	secret := &corev1.Secret{}
	if err := client.Get(context.TODO(), types.NamespacedName{Namespace: s.Namespace, Name: s.Name}, secret); err != nil {
		return nil, fmt.Errorf("unable to get secret: %s", err.Error())
	}
	cred := &awsAccountCredentials{}
	if err := json.Unmarshal(secret.Data[s.Key], cred); err != nil {
		return nil, fmt.Errorf("unable to parse secret data: %s", err.Error())
	}
	return cred, nil
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
