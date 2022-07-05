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
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"

	"antrea.io/antreacloud/apis/crd/v1alpha1"
)

// aws instance resource filter keys.
const (
	awsFilterKeyVPCID         = "vpc-id"
	awsFilterKeyVMID          = "instance-id"
	awsFilterKeyVMName        = "tag:Name"
	awsFilterKeyGroupName     = "group-name"
	awsFilterKeyInstanceState = "instance-state-code"

	// Not supported by aws, internal use only.
	awsCustomFilterKeyVPCName = "vpc-name"
)

var (
	awsInstanceStateRunningCode      = "16"
	awsInstanceStateShuttingDownCode = "32"
	awsInstanceStateStoppingCode     = "64"
	awsInstanceStateStoppedCode      = "80"
)

// convertSelectorToEC2InstanceFilters converts vm selector to aws filters.
func convertSelectorToEC2InstanceFilters(selector *v1alpha1.CloudEntitySelector) ([][]*ec2.Filter, bool) {
	if selector == nil || selector.Spec.VMSelector == nil {
		return nil, false
	}
	if selector.Spec.VMSelector.VMMatches == nil {
		return nil, true
	}

	return buildEc2Filters(selector.Spec.VMSelector.VMMatches), true
}

// buildEc2Filters builds ec2 filters for VirtualMachineMatches.
func buildEc2Filters(vmMatches []v1alpha1.VirtualMachineMatch) [][]*ec2.Filter {
	vpcIDsWithVpcIDOnlyMatches := make(map[string]struct{})
	var vpcIDWithOtherMatches []v1alpha1.VirtualMachineMatch
	var vmIDOnlyMatches []v1alpha1.VirtualMachineMatch
	var vmIDAndVMNameMatches []v1alpha1.VirtualMachineMatch
	var vmNameOnlyMatches []v1alpha1.VirtualMachineMatch
	var vpcNameOnlyMatches []v1alpha1.VirtualMachineMatch

	for _, match := range vmMatches {
		isVpcIDPresent := false
		isVMIDPresent := false
		isVMNamePresent := false
		isVpcNamePresent := false

		networkMatch := match.VpcMatch
		if networkMatch != nil {
			if len(strings.TrimSpace(networkMatch.MatchID)) > 0 {
				isVpcIDPresent = true
			}
			if len(strings.TrimSpace(networkMatch.MatchName)) > 0 {
				isVpcNamePresent = true
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

		// select all entry found. No need to process any other matches.
		if !isVpcIDPresent && !isVMIDPresent && !isVMNamePresent && !isVpcNamePresent {
			return nil
		}

		// select all for a vpc ID entry found. keep track of these vpc IDs and skip any other matches with these vpc IDs
		// as match-all overrides any specific (vmID or vmName based) matches.
		if isVpcIDPresent && !isVMIDPresent && !isVMNamePresent {
			vpcIDsWithVpcIDOnlyMatches[networkMatch.MatchID] = struct{}{}
		}

		if isVpcIDPresent && (isVMIDPresent || isVMNamePresent) {
			if _, found := vpcIDsWithVpcIDOnlyMatches[networkMatch.MatchID]; found {
				continue
			}
			vpcIDWithOtherMatches = append(vpcIDWithOtherMatches, match)
		}

		// vm id only matches.
		if isVMIDPresent && !isVMNamePresent && !isVpcIDPresent {
			vmIDOnlyMatches = append(vmIDOnlyMatches, match)
		}

		// vm id and vm name matches.
		if isVMIDPresent && isVMNamePresent && !isVpcIDPresent {
			vmIDAndVMNameMatches = append(vmIDAndVMNameMatches, match)
		}

		// vm name only matches.
		if isVMNamePresent && !isVMIDPresent && !isVpcIDPresent {
			vmNameOnlyMatches = append(vmNameOnlyMatches, match)
		}

		// vpc name only matches.
		if isVpcNamePresent && !isVMIDPresent && !isVpcIDPresent && !isVMNamePresent {
			vpcNameOnlyMatches = append(vpcNameOnlyMatches, match)
		}
	}

	awsPluginLogger().Info("selector stats", "VpcIdOnlyMatch", len(vpcIDsWithVpcIDOnlyMatches),
		"VpcIdWithOtherMatches", len(vpcIDWithOtherMatches), "VmIdOnlyMatches", len(vmIDOnlyMatches),
		"VmIdAndVmNameMatches", len(vmIDAndVMNameMatches), "VmNameOnlyMatches", len(vmNameOnlyMatches),
		"VpcNameOnlyMatches", len(vpcNameOnlyMatches))

	var allEc2Filters [][]*ec2.Filter

	vpcIDOnlyEc2Filter := buildAwsEc2FilterForVpcIDOnlyMatches(vpcIDsWithVpcIDOnlyMatches)
	if vpcIDOnlyEc2Filter != nil {
		vpcIDOnlyEc2Filter = append(vpcIDOnlyEc2Filter, buildEc2FilterForValidInstanceStates())
		allEc2Filters = append(allEc2Filters, vpcIDOnlyEc2Filter)
	}

	vpcIDWithOtherEc2Filter := buildAwsEc2FilterForVpcIDWithOtherMatches(vpcIDWithOtherMatches, vpcIDsWithVpcIDOnlyMatches)
	if vpcIDWithOtherEc2Filter != nil {
		allEc2Filters = append(allEc2Filters, vpcIDWithOtherEc2Filter...)
	}

	vmIDOnlyEc2Filter := buildAwsEc2FilterForVMIDOnlyMatches(vmIDOnlyMatches)
	if vmIDOnlyEc2Filter != nil {
		allEc2Filters = append(allEc2Filters, vmIDOnlyEc2Filter)
	}

	vmIDAndVMNameEc2Filer := buildAwsEc2FilterForVMIDAndVMNameMatches(vmIDAndVMNameMatches)
	if vmIDAndVMNameEc2Filer != nil {
		allEc2Filters = append(allEc2Filters, vmIDAndVMNameEc2Filer...)
	}

	vmNameOnlyEc2Filter := buildAwsEc2FilterForVMNameOnlyMatches(vmNameOnlyMatches)
	if vmNameOnlyEc2Filter != nil {
		allEc2Filters = append(allEc2Filters, vmNameOnlyEc2Filter)
	}

	vpcNameOnlyEc2Filter := buildAwsEc2FilterForVPCNameOnlyMatches(vpcNameOnlyMatches)
	if vpcNameOnlyEc2Filter != nil {
		allEc2Filters = append(allEc2Filters, vpcNameOnlyEc2Filter)
	}
	return allEc2Filters
}

func buildAwsEc2FilterForVpcIDOnlyMatches(vpcIDsWithVpcIDOnlyMatches map[string]struct{}) []*ec2.Filter {
	if len(vpcIDsWithVpcIDOnlyMatches) == 0 {
		return nil
	}

	var filters []*ec2.Filter
	var vpcIDs []*string

	for vpcID := range vpcIDsWithVpcIDOnlyMatches {
		vpcIDs = append(vpcIDs, aws.String(vpcID))
	}

	sort.Slice(vpcIDs, func(i, j int) bool {
		return strings.Compare(*vpcIDs[i], *vpcIDs[j]) < 0
	})
	filter := &ec2.Filter{
		Name:   aws.String(awsFilterKeyVPCID),
		Values: vpcIDs,
	}
	filters = append(filters, filter)

	return filters
}

func buildAwsEc2FilterForVpcIDWithOtherMatches(vpcIDWithOtherMatches []v1alpha1.VirtualMachineMatch,
	vpcIDsWithVpcIDOnlyMatches map[string]struct{}) [][]*ec2.Filter {
	if len(vpcIDWithOtherMatches) == 0 {
		return nil
	}

	var allFilters [][]*ec2.Filter
	for _, match := range vpcIDWithOtherMatches {
		var filters []*ec2.Filter

		vpcID := match.VpcMatch.MatchID
		if _, found := vpcIDsWithVpcIDOnlyMatches[vpcID]; found {
			continue
		}

		vpcIDFtiler := &ec2.Filter{
			Name:   aws.String(awsFilterKeyVPCID),
			Values: []*string{aws.String(vpcID)},
		}
		filters = append(filters, vpcIDFtiler)

		vmID := match.VMMatch.MatchID
		if len(strings.TrimSpace(vmID)) > 0 {
			vmIDsFilter := &ec2.Filter{
				Name:   aws.String(awsFilterKeyVMID),
				Values: []*string{aws.String(vmID)},
			}
			filters = append(filters, vmIDsFilter)
		}

		vmName := match.VMMatch.MatchName
		if len(strings.TrimSpace(vmName)) > 0 {
			vmNamesFilter := &ec2.Filter{
				Name:   aws.String(awsFilterKeyVMName),
				Values: []*string{aws.String(vmName)},
			}
			filters = append(filters, vmNamesFilter)
		}

		filters = append(filters, buildEc2FilterForValidInstanceStates())

		allFilters = append(allFilters, filters)
	}
	return allFilters
}

func buildAwsEc2FilterForVMIDOnlyMatches(vmIDOnlyMatches []v1alpha1.VirtualMachineMatch) []*ec2.Filter {
	if len(vmIDOnlyMatches) == 0 {
		return nil
	}

	var filters []*ec2.Filter
	var vmIDs []*string

	for _, match := range vmIDOnlyMatches {
		vmIDs = append(vmIDs, aws.String(match.VMMatch.MatchID))
	}

	sort.Slice(vmIDs, func(i, j int) bool {
		return strings.Compare(*vmIDs[i], *vmIDs[j]) < 0
	})
	filter := &ec2.Filter{
		Name:   aws.String(awsFilterKeyVMID),
		Values: vmIDs,
	}
	filters = append(filters, filter)
	filters = append(filters, buildEc2FilterForValidInstanceStates())

	return filters
}

func buildAwsEc2FilterForVMIDAndVMNameMatches(vmIDAndVMNameMatches []v1alpha1.VirtualMachineMatch) [][]*ec2.Filter {
	if len(vmIDAndVMNameMatches) == 0 {
		return nil
	}

	var allFilters [][]*ec2.Filter
	for _, match := range vmIDAndVMNameMatches {
		var filters []*ec2.Filter

		vmID := match.VMMatch.MatchID
		vmIDsFilter := &ec2.Filter{
			Name:   aws.String(awsFilterKeyVMID),
			Values: []*string{aws.String(vmID)},
		}
		filters = append(filters, vmIDsFilter)

		vmName := match.VMMatch.MatchName
		vmNamesFilter := &ec2.Filter{
			Name:   aws.String(awsFilterKeyVMName),
			Values: []*string{aws.String(vmName)},
		}
		filters = append(filters, vmNamesFilter)
		filters = append(filters, buildEc2FilterForValidInstanceStates())

		allFilters = append(allFilters, filters)
	}
	return allFilters
}

func buildAwsEc2FilterForVMNameOnlyMatches(vmNameOnlyMatches []v1alpha1.VirtualMachineMatch) []*ec2.Filter {
	if len(vmNameOnlyMatches) == 0 {
		return nil
	}

	var filters []*ec2.Filter
	var vmNames []*string

	for _, match := range vmNameOnlyMatches {
		vmNames = append(vmNames, aws.String(match.VMMatch.MatchName))
	}

	sort.Slice(vmNames, func(i, j int) bool {
		return strings.Compare(*vmNames[i], *vmNames[j]) < 0
	})
	filter := &ec2.Filter{
		Name:   aws.String(awsFilterKeyVMName),
		Values: vmNames,
	}
	filters = append(filters, filter)
	filters = append(filters, buildEc2FilterForValidInstanceStates())

	return filters
}

func buildAwsEc2FilterForVPCNameOnlyMatches(vpcNameOnlyMatches []v1alpha1.VirtualMachineMatch) []*ec2.Filter {
	if len(vpcNameOnlyMatches) == 0 {
		return nil
	}

	var filters []*ec2.Filter
	var vpcNames []*string

	for _, match := range vpcNameOnlyMatches {
		vpcNames = append(vpcNames, aws.String(match.VpcMatch.MatchName))
	}

	sort.Slice(vpcNames, func(i, j int) bool {
		return strings.Compare(*vpcNames[i], *vpcNames[j]) < 0
	})
	filter := &ec2.Filter{
		Name:   aws.String(awsCustomFilterKeyVPCName),
		Values: vpcNames,
	}
	filters = append(filters, filter)
	filters = append(filters, buildEc2FilterForValidInstanceStates())

	return filters
}

func buildFilterForVPCIDFromFilterForVPCName(filtersForVPCName []*ec2.Filter, vpcNameToID map[string]string) []*ec2.Filter {
	if len(filtersForVPCName) == 0 {
		return nil
	}

	var filters []*ec2.Filter
	var vpcIDs []*string

	for _, filter := range filtersForVPCName {
		if *filter.Name != awsFilterKeyInstanceState {
			for _, vpcName := range filter.Values {
				vpcIDs = append(vpcIDs, aws.String(vpcNameToID[*vpcName]))
			}
		}
	}

	sort.Slice(vpcIDs, func(i, j int) bool {
		return strings.Compare(*vpcIDs[i], *vpcIDs[j]) < 0
	})
	filter := &ec2.Filter{
		Name:   aws.String(awsFilterKeyVPCID),
		Values: vpcIDs,
	}
	filters = append(filters, filter)
	filters = append(filters, buildEc2FilterForValidInstanceStates())

	return filters
}

func buildAwsEc2FilterForSecurityGroupNameMatches(vpcIDsSet []string, cloudSGNamesSet map[string]struct{}) []*ec2.Filter {
	var filters []*ec2.Filter
	var vpcIDs []*string
	var cloudSGNames []*string

	for _, id := range vpcIDsSet {
		idCopy := aws.String(id)
		vpcIDs = append(vpcIDs, idCopy)
	}

	for name := range cloudSGNamesSet {
		nameCopy := name
		cloudSGNames = append(cloudSGNames, &nameCopy)
	}

	vpcIDFilter := &ec2.Filter{
		Name:   aws.String(awsFilterKeyVPCID),
		Values: vpcIDs,
	}
	securityGroupNamesFilter := &ec2.Filter{
		Name:   aws.String(awsFilterKeyGroupName),
		Values: cloudSGNames,
	}
	filters = append(filters, vpcIDFilter)
	filters = append(filters, securityGroupNamesFilter)

	return filters
}

func buildEc2FilterForValidInstanceStates() *ec2.Filter {
	states := []*string{&awsInstanceStateRunningCode, &awsInstanceStateShuttingDownCode, &awsInstanceStateStoppedCode,
		&awsInstanceStateStoppingCode}

	instanceStateFilter := &ec2.Filter{
		Name:   aws.String(awsFilterKeyInstanceState),
		Values: states,
	}

	return instanceStateFilter
}
