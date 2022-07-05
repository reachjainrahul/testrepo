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

package securitygroup

import (
	"strings"
)

func IsAntreaCloudCreatedSecurityGroup(cloudSgName string) (string, bool, bool) {
	var antreaCloudName string
	isAntreaCloudCreatedAddressGroup := false
	isAntreaCloudCreatedAppliedToGroup := false

	suffix := strings.TrimPrefix(cloudSgName, AntreaCloudAddressGroupPrefix)
	if len(suffix) < len(cloudSgName) {
		isAntreaCloudCreatedAddressGroup = true
		antreaCloudName = strings.ToLower(suffix)
	}

	if !isAntreaCloudCreatedAddressGroup {
		suffix := strings.TrimPrefix(cloudSgName, AntreaCloudAppliedToPrefix)
		if len(suffix) < len(cloudSgName) {
			isAntreaCloudCreatedAppliedToGroup = true
			antreaCloudName = strings.ToLower(suffix)
		}
	}
	return antreaCloudName, isAntreaCloudCreatedAddressGroup, isAntreaCloudCreatedAppliedToGroup
}

func FindResourcesBasedOnKind(cloudResources []*CloudResource) (map[string]struct{}, map[string]struct{}) {
	virtualMachineIDs := make(map[string]struct{})
	networkInterfaceIDs := make(map[string]struct{})

	for _, cloudResource := range cloudResources {
		if strings.Compare(string(cloudResource.Type), string(CloudResourceTypeVM)) == 0 {
			virtualMachineIDs[strings.ToLower(cloudResource.Name.Name)] = struct{}{}
		}
		if strings.Compare(string(cloudResource.Type), string(CloudResourceTypeNIC)) == 0 {
			networkInterfaceIDs[strings.ToLower(cloudResource.Name.Name)] = struct{}{}
		}
	}
	return virtualMachineIDs, networkInterfaceIDs
}
