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

package cloud

import (
	v1alpha1 "antrea.io/nephe/apis/crd/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var (
	testAccountNamespacedName = types.NamespacedName{Namespace: "namespace01", Name: "account01"}
)
var _ = Describe("Cloudprovideraccount", func() {

	var (
		accountAWS   *v1alpha1.CloudProviderAccount
		accountAzure *v1alpha1.CloudProviderAccount
	)

	BeforeEach(func() {
		var pollIntv uint = 30
		accountAWS = &v1alpha1.CloudProviderAccount{
			ObjectMeta: v1.ObjectMeta{
				Name:      testAccountNamespacedName.Name,
				Namespace: testAccountNamespacedName.Namespace,
			},
			Spec: v1alpha1.CloudProviderAccountSpec{
				PollIntervalInSeconds: &pollIntv,
				AWSConfig: &v1alpha1.CloudProviderAccountAWSConfig{
					AccountID: "TestAccount01",
					//AccessKeyID:     "keyId",
					//AccessKeySecret: "keySecret",
					Region: "us-east-1",
					//RoleArn: "testArn",
				},
			},
		}
		accountAzure = &v1alpha1.CloudProviderAccount{
			ObjectMeta: v1.ObjectMeta{
				Name:      testAccountNamespacedName.Name,
				Namespace: testAccountNamespacedName.Namespace,
			},
			Spec: v1alpha1.CloudProviderAccountSpec{
				PollIntervalInSeconds: &pollIntv,
				AzureConfig: &v1alpha1.CloudProviderAccountAzureConfig{
					//SubscriptionID:   "SubID",
					//ClientID:         "ClientID",
					//TenantID:         "TenantID",
					//ClientKey:        "ClientKey",
					Region: "eastus",
				},
			},
		}
	})
	Context("New AWS account add fail scenarios", func() {
		It("Should fail with empty account ID", func() {
			accountAWS.Spec.AWSConfig.AccountID = ""

			err := accountAWS.ValidateCreate()
			Expect(err).ShouldNot(BeNil())
		})

		It("Should fail with blank account ID", func() {
			accountAWS.Spec.AWSConfig.AccountID = "			"

			err := accountAWS.ValidateCreate()
			Expect(err).ShouldNot(BeNil())
		})

		It("Should fail for with empty credential and roleArn", func() {
			//accountAWS.Spec.AWSConfig.AccessKeyID = ""
			//accountAWS.Spec.AWSConfig.AccessKeySecret = ""
			//accountAWS.Spec.AWSConfig.RoleArn = ""

			err := accountAWS.ValidateCreate()
			Expect(err).ShouldNot(BeNil())
		})

		It("Should fail for with blank credential and roleArn", func() {
			//accountAWS.Spec.AWSConfig.AccessKeyID = "			"
			//accountAWS.Spec.AWSConfig.AccessKeySecret = "			"
			//accountAWS.Spec.AWSConfig.RoleArn = "			"

			err := accountAWS.ValidateCreate()
			Expect(err).ShouldNot(BeNil())
		})

		It("Should validate AWS account successfully", func() {

			err := accountAWS.ValidateCreate()
			Expect(err).Should(BeNil())
		})
	})

	Context("New Azure account add fail scenarios", func() {
		It("Should fail with empty subscription ID", func() {
			//accountAzure.Spec.AzureConfig.SubscriptionID = ""

			err := accountAzure.ValidateCreate()
			Expect(err).ShouldNot(BeNil())
		})

		It("Should fail with blank subscription ID", func() {
			//accountAzure.Spec.AzureConfig.SubscriptionID = "			"

			err := accountAzure.ValidateCreate()
			Expect(err).ShouldNot(BeNil())
		})

		It("Should fail for with empty credential and identity", func() {
			//accountAzure.Spec.AzureConfig.ClientID = ""
			//accountAzure.Spec.AzureConfig.ClientKey = ""

			err := accountAzure.ValidateCreate()
			Expect(err).ShouldNot(BeNil())
		})

		It("Should fail for with blank credential and identity", func() {
			//accountAzure.Spec.AzureConfig.ClientID = "			"
			//accountAzure.Spec.AzureConfig.ClientKey = "			"

			err := accountAzure.ValidateCreate()
			Expect(err).ShouldNot(BeNil())
		})

		It("Should validate Azure account successfully", func() {
			err := accountAzure.ValidateCreate()
			Expect(err).Should(BeNil())
		})
	})
})
