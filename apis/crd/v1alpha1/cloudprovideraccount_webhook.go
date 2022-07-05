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
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var cloudprovideraccountlog = logf.Log.WithName("cloudprovideraccount-resource")

func (r *CloudProviderAccount) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// nolint:lll
// +kubebuilder:webhook:path=/mutate-crd-cloud-antrea-io-v1alpha1-cloudprovideraccount,mutating=true,failurePolicy=fail,groups=crd.cloud.antrea.io,resources=cloudprovideraccounts,verbs=create,versions=v1alpha1,name=mcloudprovideraccount.kb.io,sideEffects=None,admissionReviewVersions=v1;v1beta1

var _ webhook.Defaulter = &CloudProviderAccount{}

// Default implements webhook.Defaulter so a webhook will be registered for the type.
func (r *CloudProviderAccount) Default() {
	cloudprovideraccountlog.Info("default", "name", r.Name)

	if r.Spec.PollIntervalInSeconds == nil {
		var defaultIntv uint = 60
		r.Spec.PollIntervalInSeconds = &defaultIntv
	}
}

// TODO(user): change verbs to :"verbs=create;update;delete" if you want to enable deletion validation.
// nolint:lll
// +kubebuilder:webhook:verbs=create,path=/validate-crd-cloud-antrea-io-v1alpha1-cloudprovideraccount,mutating=false,failurePolicy=fail,groups=crd.cloud.antrea.io,resources=cloudprovideraccounts,versions=v1alpha1,name=vcloudprovideraccount.kb.io,sideEffects=None,admissionReviewVersions=v1;v1beta1

var _ webhook.Validator = &CloudProviderAccount{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (r *CloudProviderAccount) ValidateCreate() error {
	cloudprovideraccountlog.Info("validate create", "name", r.Name)

	cloudProviderType := r.Spec.ProviderType
	switch cloudProviderType {
	case AWSCloudProvider:
		configAWS := r.Spec.ConfigAWS
		if configAWS == nil {
			return fmt.Errorf("configAWS cannot be nil")
		}
		if len(strings.TrimSpace(configAWS.AccountID)) == 0 ||
			len(strings.TrimSpace(configAWS.Region)) == 0 {
			return fmt.Errorf("accountID and region are required fields for AWS config")
		}
	case AzureCloudProvider:
		// TODO
	default:
		return fmt.Errorf("unknown/unsupported cloud provier type %v (valid values AWS, Azure)", cloudProviderType)
	}

	if *r.Spec.PollIntervalInSeconds < 30 {
		return fmt.Errorf("pollIntervalInSeconds should be >= 30. If not specified, defaults to 60")
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (r *CloudProviderAccount) ValidateUpdate(old runtime.Object) error {
	cloudprovideraccountlog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (r *CloudProviderAccount) ValidateDelete() error {
	cloudprovideraccountlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
