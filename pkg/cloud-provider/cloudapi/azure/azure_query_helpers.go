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
	"sort"
	"strings"

	"antrea.io/antreacloud/apis/crd/v1alpha1"
)

func convertSelectorToComputeQuery(selector *v1alpha1.CloudEntitySelector, subscriptionIDs []string,
	tenantIDs []string, locations []string) ([]*string, bool) {
	if selector == nil || selector.Spec.VMSelector == nil {
		return nil, false
	}
	if selector.Spec.VMSelector.VMMatches == nil {
		return nil, true
	}

	allQueryStrings, err := buildQueries(selector.Spec.VMSelector.VMMatches, subscriptionIDs, tenantIDs, locations)
	if err != nil {
		azurePluginLogger().Error(err, "selector conversion to query failed",
			"selectorName", selector.Name, "selectorNamespace", selector.Namespace)
		return nil, false
	}

	return allQueryStrings, true
}

func buildQueries(vmMatches []v1alpha1.VirtualMachineMatch, subscriptionIDs []string, tenantIDs []string,
	locations []string) ([]*string, error) {
	vpcIDsWithVpcIDOnlyMatches := make(map[string]struct{})
	var vpcIDWithOtherMatches []v1alpha1.VirtualMachineMatch
	var vmIDOnlyMatches []v1alpha1.VirtualMachineMatch
	var vmIDAndVMNameMatches []v1alpha1.VirtualMachineMatch
	var vmNameOnlyMatches []v1alpha1.VirtualMachineMatch

	for _, match := range vmMatches {
		isVpcIDPresent := false
		isVMIDPresent := false
		isVMNamePresent := false

		networkMatch := match.VpcMatch
		if networkMatch != nil {
			if len(strings.TrimSpace(networkMatch.MatchID)) > 0 {
				isVpcIDPresent = true
			}
		}
		vmMatch := match.VMMatch
		if vmMatch != nil {
			if len(strings.TrimSpace(vmMatch.MatchID)) > 0 {
				isVMIDPresent = true
			}
			if len(strings.TrimSpace(vmMatch.MatchName)) > 0 {
				isVMNamePresent = true
			}
		}

		// select all entry found. No need to process any other matches
		if !isVpcIDPresent && !isVMIDPresent && !isVMNamePresent {
			return nil, nil
		}

		// select all for a vpc ID entry found. keep track of these vpc IDs and skip any other matches with these vpc IDs
		// as match-all overrides any specific (vmID or vmName based) matches
		if isVpcIDPresent && !isVMIDPresent && !isVMNamePresent {
			vpcIDsWithVpcIDOnlyMatches[networkMatch.MatchID] = struct{}{}
		}

		if isVpcIDPresent && (isVMIDPresent || isVMNamePresent) {
			if _, found := vpcIDsWithVpcIDOnlyMatches[networkMatch.MatchID]; found {
				continue
			}
			vpcIDWithOtherMatches = append(vpcIDWithOtherMatches, match)
		}

		// vm id only matches
		if isVMIDPresent && !isVMNamePresent && !isVpcIDPresent {
			vmIDOnlyMatches = append(vmIDOnlyMatches, match)
		}

		// vm id and vm name matches
		if isVMIDPresent && isVMNamePresent && !isVpcIDPresent {
			vmIDAndVMNameMatches = append(vmIDAndVMNameMatches, match)
		}

		// vm name only matches
		if isVMNamePresent && !isVMIDPresent && !isVpcIDPresent {
			vmNameOnlyMatches = append(vmNameOnlyMatches, match)
		}
	}

	azurePluginLogger().Info("selector stats", "VpcIdOnlyMatch", len(vpcIDsWithVpcIDOnlyMatches),
		"VpcIdWithOtherMatches", len(vpcIDWithOtherMatches), "VmIdOnlyMatches", len(vmIDOnlyMatches),
		"VmIdAndVmNameMatches", len(vmIDAndVMNameMatches), "VmNameOnlyMatches", len(vmNameOnlyMatches))

	var allQueries []*string

	vpcIDOnlyQuery, err := buildQueryForVpcIDOnlyMatches(vpcIDsWithVpcIDOnlyMatches, subscriptionIDs, tenantIDs, locations)
	if err != nil {
		return nil, err
	}
	if vpcIDOnlyQuery != nil {
		allQueries = append(allQueries, vpcIDOnlyQuery)
	}

	vmNameOnlyQuery, err := buildQueryForVMNameOnlyMatches(vmNameOnlyMatches, subscriptionIDs, tenantIDs, locations)
	if err != nil {
		return nil, err
	}
	if vmNameOnlyQuery != nil {
		allQueries = append(allQueries, vmNameOnlyQuery)
	}

	vmIDOnlyQuery, err := buildQueryForVMIDOnlyMatches(vmIDOnlyMatches, subscriptionIDs, tenantIDs, locations)
	if err != nil {
		return nil, err
	}
	if vmIDOnlyQuery != nil {
		allQueries = append(allQueries, vmIDOnlyQuery)
	}

	return allQueries, nil
}

func buildQueryForVpcIDOnlyMatches(vpcIDsWithVpcIDOnlyMatches map[string]struct{}, subscriptionIDs []string, tenantIDs []string,
	locations []string) (*string, error) {
	if len(vpcIDsWithVpcIDOnlyMatches) == 0 {
		return nil, nil
	}

	var vpcIDs []string

	for vpcID := range vpcIDsWithVpcIDOnlyMatches {
		vpcIDs = append(vpcIDs, vpcID)
	}

	sort.Slice(vpcIDs, func(i, j int) bool {
		return strings.Compare(vpcIDs[i], vpcIDs[j]) < 0
	})

	return getVMsByVnetIDsAndSubscriptionIDsAndTenantIDsAndLocationsMatchQuery(vpcIDs, subscriptionIDs, tenantIDs, locations)
}

func buildQueryForVMNameOnlyMatches(vmNameOnlyMatches []v1alpha1.VirtualMachineMatch, subscriptionIDs []string, tenantIDs []string,
	locations []string) (*string, error) {
	if len(vmNameOnlyMatches) == 0 {
		return nil, nil
	}

	var vmNames []string

	for _, match := range vmNameOnlyMatches {
		vmNames = append(vmNames, match.VMMatch.MatchName)
	}

	sort.Slice(vmNames, func(i, j int) bool {
		return strings.Compare(vmNames[i], vmNames[j]) < 0
	})

	return getVMsByVMNamesAndSubscriptionIDsAndTenantIDsAndLocationsMatchQuery(vmNames, subscriptionIDs, tenantIDs, locations)
}

func buildQueryForVMIDOnlyMatches(vmIDOnlyMatches []v1alpha1.VirtualMachineMatch, subscriptionIDs []string, tenantIDs []string,
	locations []string) (*string, error) {
	if len(vmIDOnlyMatches) == 0 {
		return nil, nil
	}

	var vmIDs []string

	for _, match := range vmIDOnlyMatches {
		vmIDs = append(vmIDs, match.VMMatch.MatchID)
	}

	sort.Slice(vmIDs, func(i, j int) bool {
		return strings.Compare(vmIDs[i], vmIDs[j]) < 0
	})

	return getVMsByVMIDsAndSubscriptionIDsAndTenantIDsAndLocationsMatchQuery(vmIDs, subscriptionIDs, tenantIDs, locations)
}
