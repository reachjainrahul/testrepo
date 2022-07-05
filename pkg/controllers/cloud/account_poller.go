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
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-logr/logr"
	"go.uber.org/multierr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	cloudv1alpha1 "antrea.io/antreacloud/apis/crd/v1alpha1"
	cloudprovider "antrea.io/antreacloud/pkg/cloud-provider"
	"antrea.io/antreacloud/pkg/cloud-provider/cloudapi/common"
)

const (
	accountResourceToCreate = "TO_CREATE"
	accountResourceToDelete = "TO_DELETE"
	accountResourceToUpdate = "TO_UPDATE"
)

type accountPoller struct {
	client.Client
	log    logr.Logger
	scheme *runtime.Scheme

	pollIntvInSeconds uint
	cloudType         cloudv1alpha1.CloudProvider
	namespacedName    *types.NamespacedName
	selector          *cloudv1alpha1.CloudEntitySelector
	ch                chan struct{}
	cloudInventory    *CloudInventory
	vmMatches         cache.Indexer
	vmMatchKey        uint64
}

func (p *accountPoller) doAccountPoller() {
	cloudInterface, e := cloudprovider.GetCloudInterface(common.ProviderType(p.cloudType))
	if e != nil {
		p.log.Info("failed to get cloud interface", "account", p.namespacedName, "error", e)
		return
	}

	account := &cloudv1alpha1.CloudProviderAccount{}
	e = p.Get(context.TODO(), *p.namespacedName, account)
	if e != nil {
		p.log.Info("failed to get account", "account", p.namespacedName, "account", account, "error", e)
	}

	discoveredstatus, e := cloudInterface.GetAccountStatus(p.namespacedName)
	if e != nil {
		p.log.Info("failed to get account status", "account", p.namespacedName, "error", e)
	} else {
		updateAccountStatus(&account.Status, discoveredstatus)
	}

	e = p.Client.Status().Update(context.TODO(), account)
	if e != nil {
		p.log.Info("failed to update account status", "account", p.namespacedName, "err", e)
	}
	virtualMachines, vmNetworkInterfaces := p.getComputeResources(cloudInterface)

	e = p.doVirtualMachineOperations(virtualMachines)
	if e != nil {
		p.log.Info("failed to perform virtual-machine operations", "account", p.namespacedName, "error", e)
	}

	var networkInterfaces []*cloudv1alpha1.NetworkInterface
	networkInterfaces = append(networkInterfaces, vmNetworkInterfaces...)
	e = p.doNetworkInterfaceOperations(networkInterfaces)
	if e != nil {
		p.log.Info("failed to perform network-interface operations", "account", p.namespacedName, "error", e)
	}
}

func (p *accountPoller) getComputeResources(cloudInterface common.CloudInterface) ([]*cloudv1alpha1.VirtualMachine,
	[]*cloudv1alpha1.NetworkInterface) {
	var e error

	virtualMachines, networkInterfaces, e := cloudInterface.InstancesGivenProviderAccount(p.namespacedName)
	if e != nil {
		p.log.Info("failed to discover compute resources", "account", p.namespacedName, "error", e)
		return []*cloudv1alpha1.VirtualMachine{}, []*cloudv1alpha1.NetworkInterface{}
	}

	p.log.Info("discovered compute resources statistics", "account", p.namespacedName, "virtual-machines",
		len(virtualMachines), "network-interfaces", len(networkInterfaces))

	return virtualMachines, networkInterfaces
}

func (p *accountPoller) doVirtualMachineOperations(virtualMachines []*cloudv1alpha1.VirtualMachine) error {
	virtualMachinesBasedOnOperation, err := p.findVirtualMachinesByOperation(virtualMachines)
	if err != nil {
		return err
	}

	virtualMachinesToCreate, found := virtualMachinesBasedOnOperation[accountResourceToCreate]
	if found {
		for _, vm := range virtualMachinesToCreate {
			e := controllerutil.SetControllerReference(p.selector, vm, p.scheme)
			if e != nil {
				p.log.Info("error setting controller owner reference", "err", e)
				err = multierr.Append(err, e)
				continue
			}
			// save status since Create will update vm object and remove status field from it
			vmStatus := vm.Status
			e = p.Client.Create(context.TODO(), vm)
			if e != nil {
				p.log.Info("virtual machine create failed", "name", vm.Name, "err", e)
				err = multierr.Append(err, e)
				continue
			}
			vm.Status = vmStatus
			e = p.Client.Status().Update(context.TODO(), vm)
			if e != nil {
				p.log.Info("virtual machine status update failed", "account", p.namespacedName, "name", vm.Name, "err", e)
				err = multierr.Append(err, e)
				continue
			}
			p.cloudInventory.AddVirtualMachine(vm)
		}
	}

	virtualMachinesToDelete, found := virtualMachinesBasedOnOperation[accountResourceToDelete]
	if found {
		for _, vm := range virtualMachinesToDelete {
			e := p.Delete(context.TODO(), vm)
			if e != nil {
				if client.IgnoreNotFound(e) != nil {
					err = multierr.Append(err, e)
					p.log.Info("unable to delete", "vm-name", vm.Name)
					continue
				}
			}
			p.log.Info("deleted", "vm-name", vm.Name)
			p.cloudInventory.DeleteVirtualMachine(vm)
		}
	}

	virtualMachinesToUpdate, found := virtualMachinesBasedOnOperation[accountResourceToUpdate]
	if found {
		for _, vm := range virtualMachinesToUpdate {
			vmNamespacedName := types.NamespacedName{
				Namespace: vm.Namespace,
				Name:      vm.Name,
			}
			currentVM := &cloudv1alpha1.VirtualMachine{}
			e := p.Get(context.TODO(), vmNamespacedName, currentVM)
			if e != nil {
				if client.IgnoreNotFound(e) != nil {
					err = multierr.Append(err, e)
					p.log.Info("unable to find to update", "vm-name", vm.Name)
					continue
				}
			}

			updateCloudDiscoveredFieldsOfVirtualMachineStatus(&currentVM.Status, &vm.Status)
			e = p.Client.Status().Update(context.TODO(), currentVM)
			if e != nil {
				p.log.Info("virtual machine status update failed", "account", p.namespacedName, "name", vm.Name, "err", e)
				err = multierr.Append(err, e)
				continue
			}
			p.log.Info("updated", "vm-name", vm.Name)
			p.cloudInventory.UpdateVirtualMachine(vm)
		}
	}

	p.log.Info("virtual-machine crd statistics", "account", p.namespacedName,
		"created", len(virtualMachinesToCreate), "deleted", len(virtualMachinesToDelete), "updated", len(virtualMachinesToUpdate))

	return err
}

func (p *accountPoller) findVirtualMachinesByOperation(discoveredVirtualMachines []*cloudv1alpha1.VirtualMachine) (
	map[string][]*cloudv1alpha1.VirtualMachine, error) {
	virtualMachinesByOperation := make(map[string][]*cloudv1alpha1.VirtualMachine)

	currentVirtualMachinesByName, err := p.getCurrentVirtualMachinesByName()
	if err != nil {
		return nil, err
	}

	// if no virtual machines in etcd, all discovered needs to be created.
	if len(currentVirtualMachinesByName) == 0 {
		virtualMachinesByOperation[accountResourceToCreate] = discoveredVirtualMachines
		return virtualMachinesByOperation, nil
	}

	// find virtual machines to be created.
	// And also removed any vm which needs to be created from currentVirtualMachineByName map.
	var virtualMachinesToCreate []*cloudv1alpha1.VirtualMachine
	var virtualMachinesToUpdate []*cloudv1alpha1.VirtualMachine
	for _, discoveredVirtualMachine := range discoveredVirtualMachines {
		currentVirtualMachine, found := currentVirtualMachinesByName[discoveredVirtualMachine.Name]
		if !found {
			virtualMachinesToCreate = append(virtualMachinesToCreate, discoveredVirtualMachine)
		} else {
			delete(currentVirtualMachinesByName, currentVirtualMachine.Name)
			if !areDiscoveredFieldsSameVirtualMachineStatus(currentVirtualMachine.Status, discoveredVirtualMachine.Status) {
				virtualMachinesToUpdate = append(virtualMachinesToUpdate, discoveredVirtualMachine)
			}
		}
	}

	// find virtual machines to be deleted.
	// All entries remaining in currentVirtualMachineByName are to be deleted from etcd
	var virtualMachinesToDelete []*cloudv1alpha1.VirtualMachine
	for _, vmToDelete := range currentVirtualMachinesByName {
		virtualMachinesToDelete = append(virtualMachinesToDelete, vmToDelete.DeepCopy())
	}

	virtualMachinesByOperation[accountResourceToCreate] = virtualMachinesToCreate
	virtualMachinesByOperation[accountResourceToDelete] = virtualMachinesToDelete
	virtualMachinesByOperation[accountResourceToUpdate] = virtualMachinesToUpdate

	return virtualMachinesByOperation, nil
}

func (p *accountPoller) doNetworkInterfaceOperations(networkInterfaces []*cloudv1alpha1.NetworkInterface) error {
	networkInterfacesBasedOnOperation, err := p.findNetworkInterfacesByOperation(networkInterfaces)
	if err != nil {
		return err
	}

	networkInterfacesToCreate, found := networkInterfacesBasedOnOperation[accountResourceToCreate]
	if found {
		for _, intf := range networkInterfacesToCreate {
			// save status since Create will update intf object and remove status field from it
			intfStatus := intf.Status
			e := p.Client.Create(context.TODO(), intf)
			if e != nil {
				p.log.Info("network-interface create failed", "name", intf.Name, "err", e)
				err = multierr.Append(err, e)
				continue
			}
			intf.Status = intfStatus
			e = p.Client.Status().Update(context.TODO(), intf)
			if e != nil {
				p.log.Info("network-interface status update failed", "name", intf.Name, "err", e)
				err = multierr.Append(err, e)
				continue
			}
			p.cloudInventory.AddNetworkInterface(intf)
		}
	}

	networkInterfacesToDelete, found := networkInterfacesBasedOnOperation[accountResourceToDelete]
	if found {
		for _, nic := range networkInterfacesToDelete {
			e := p.Delete(context.TODO(), nic)
			if client.IgnoreNotFound(e) != nil {
				err = multierr.Append(err, e)
				p.log.Info("unable to delete", "networkInterface-name", nic.Name)
				continue
			}
			p.log.Info("deleted", "networkInterface-name", nic.Name)
			p.cloudInventory.DeleteNetworkInterface(nic)
		}
	}

	networkInterfacesToUpdate, found := networkInterfacesBasedOnOperation[accountResourceToUpdate]
	if found {
		for _, nic := range networkInterfacesToUpdate {
			nwIntfNamespacedName := types.NamespacedName{
				Namespace: nic.Namespace,
				Name:      nic.Name,
			}
			currentNwIntf := &cloudv1alpha1.NetworkInterface{}
			e := p.Get(context.TODO(), nwIntfNamespacedName, currentNwIntf)
			if e != nil {
				if client.IgnoreNotFound(e) != nil {
					err = multierr.Append(err, e)
					p.log.Info("unable to find to update", "network-interface-name", nic.Name)
					continue
				}
			}
			updateCloudDiscoveredFieldsOfNetworkInterfaceStatus(&currentNwIntf.Status, &nic.Status)
			e = p.Client.Status().Update(context.TODO(), currentNwIntf)
			if e != nil {
				p.log.Info("network-interface status update failed", "account", p.namespacedName, "name", nic.Name, "err", e)
				err = multierr.Append(err, e)
				continue
			}
			p.log.Info("updated", "vm-name", nic.Name)
			p.cloudInventory.UpdateNetworkInterface(nic)
		}
	}

	p.log.Info("network-interface crd statistics", "account", p.namespacedName,
		"created", len(networkInterfacesToCreate), "deleted", len(networkInterfacesToDelete), "updated", len(networkInterfacesToUpdate))

	return err
}

func (p *accountPoller) findNetworkInterfacesByOperation(discoveredNetworkInterfaces []*cloudv1alpha1.NetworkInterface) (
	map[string][]*cloudv1alpha1.NetworkInterface, error) {
	networkInterfacesByOperation := make(map[string][]*cloudv1alpha1.NetworkInterface)

	currentVirtualMachinesByName, err := p.getCurrentVirtualMachinesByName()
	if err != nil {
		return nil, err
	}

	currentNetworkInterfacesByName, err := p.getCurrentNetworkInterfacesByName(currentVirtualMachinesByName)
	if err != nil {
		return nil, err
	}

	// find network interfaces to be created and updated.
	var networkInterfacesToCreate []*cloudv1alpha1.NetworkInterface
	var networkInterfacesToUpdate []*cloudv1alpha1.NetworkInterface
	for _, discoveredNetworkInterface := range discoveredNetworkInterfaces {
		currentNetworkInterface, found := currentNetworkInterfacesByName[discoveredNetworkInterface.Name]
		if !found {
			// Should only be created if owner VirtualMachine exists.
			// NetworkInterface can have only one owner
			ownerReferences := discoveredNetworkInterface.GetOwnerReferences()
			if ownerReferences == nil {
				p.log.Info("no owner found. skipping create", "network-interface",
					discoveredNetworkInterface.GetName(), "namespace", discoveredNetworkInterface.GetNamespace())
				continue
			}
			ownerName := ownerReferences[0].Name

			// add to create list only if owner virtual-machine exists.
			// for any reason if virtual-machine persist failed, no need to write its network interfaces
			ownerVirtualMachine, isVMOwner := currentVirtualMachinesByName[ownerName]
			if isVMOwner {
				owner := &ownerVirtualMachine
				// update owner to the found virtual machine
				discoveredNetworkInterface.SetOwnerReferences(nil)
				err := controllerutil.SetControllerReference(owner, discoveredNetworkInterface, p.scheme)
				if err != nil {
					p.log.Info("error setting controller owner reference. skipping create", "network-interface",
						discoveredNetworkInterface.GetName(), "namespace", discoveredNetworkInterface.GetNamespace())
					continue
				}
				networkInterfacesToCreate = append(networkInterfacesToCreate, discoveredNetworkInterface)
			}
		} else {
			delete(currentNetworkInterfacesByName, currentNetworkInterface.Name)
			if !areDiscoveredFieldsSameNetworkInterfaceStatus(currentNetworkInterface.Status, discoveredNetworkInterface.Status) {
				networkInterfacesToUpdate = append(networkInterfacesToUpdate, discoveredNetworkInterface)
			}
		}
	}

	// find network interfaces to be deleted.
	var networkInterfacesToDelete []*cloudv1alpha1.NetworkInterface
	for _, intfToDelete := range currentNetworkInterfacesByName {
		// add to delete slice only if has owner and owner virtual machine exists
		ownerReferences := intfToDelete.GetOwnerReferences()
		if ownerReferences == nil {
			p.log.Info("no owner found. Skipping delete", "network-interface",
				intfToDelete.GetName(), "namespace", intfToDelete.GetNamespace())
			continue
		}
		ownerVirtualMachineName := ownerReferences[0].Name
		_, found := currentVirtualMachinesByName[ownerVirtualMachineName]
		if found {
			networkInterfacesToDelete = append(networkInterfacesToDelete, intfToDelete.DeepCopy())
		}
	}

	networkInterfacesByOperation[accountResourceToCreate] = networkInterfacesToCreate
	networkInterfacesByOperation[accountResourceToDelete] = networkInterfacesToDelete
	networkInterfacesByOperation[accountResourceToUpdate] = networkInterfacesToUpdate

	return networkInterfacesByOperation, nil
}

func (p *accountPoller) getCurrentVirtualMachinesByName() (map[string]cloudv1alpha1.VirtualMachine, error) {
	currentVirtualMachinesByName := make(map[string]cloudv1alpha1.VirtualMachine)

	currentVirtualMachineList := &cloudv1alpha1.VirtualMachineList{}
	err := p.Client.List(context.TODO(), currentVirtualMachineList, client.InNamespace(p.selector.Namespace))
	if err != nil {
		return nil, err
	}

	ownerSelector := map[string]*cloudv1alpha1.CloudEntitySelector{p.selector.Name: p.selector}
	currentVirtualMachines := currentVirtualMachineList.Items
	for _, currentVirtualMachine := range currentVirtualMachines {
		if !isVirtualMachineOwnedBy(currentVirtualMachine, ownerSelector) {
			continue
		}
		currentVirtualMachinesByName[currentVirtualMachine.Name] = currentVirtualMachine
	}
	return currentVirtualMachinesByName, nil
}

func (p *accountPoller) getCurrentNetworkInterfacesByName(currentVirtualMachines map[string]cloudv1alpha1.VirtualMachine) (
	map[string]cloudv1alpha1.NetworkInterface, error) {
	currentNetworkInterfacesByName := make(map[string]cloudv1alpha1.NetworkInterface)

	currentNetworkInterfaceList := &cloudv1alpha1.NetworkInterfaceList{}
	err := p.Client.List(context.TODO(), currentNetworkInterfaceList, client.InNamespace(p.selector.Namespace))
	if err != nil {
		return nil, err
	}

	currentNetworkInterfaces := currentNetworkInterfaceList.Items
	for _, currentNetworkInterface := range currentNetworkInterfaces {
		if !isNetworkInterfaceOwnedBy(currentNetworkInterface, currentVirtualMachines) {
			continue
		}
		currentNetworkInterfacesByName[currentNetworkInterface.Name] = currentNetworkInterface
	}

	return currentNetworkInterfacesByName, nil
}

func isVirtualMachineOwnedBy(virtualMachine cloudv1alpha1.VirtualMachine,
	ownerSelector map[string]*cloudv1alpha1.CloudEntitySelector) bool {
	vmOwnerReferences := virtualMachine.OwnerReferences
	for _, vmOwnerReference := range vmOwnerReferences {
		vmOwnerName := vmOwnerReference.Name
		vmOwnerKind := vmOwnerReference.Kind

		if _, found := ownerSelector[vmOwnerName]; found {
			if strings.Compare(vmOwnerKind, reflect.TypeOf(cloudv1alpha1.CloudEntitySelector{}).Name()) == 0 {
				return true
			}
		}
	}
	return false
}

func isNetworkInterfaceOwnedBy(networkInterface cloudv1alpha1.NetworkInterface,
	ownerVirtualMachines map[string]cloudv1alpha1.VirtualMachine) bool {
	nicOwnerReferences := networkInterface.OwnerReferences
	for _, nicOwnerReference := range nicOwnerReferences {
		nicOwnerName := nicOwnerReference.Name
		nicOwnerKind := nicOwnerReference.Kind
		if _, found := ownerVirtualMachines[nicOwnerName]; found {
			if strings.Compare(nicOwnerKind, reflect.TypeOf(cloudv1alpha1.VirtualMachine{}).Name()) == 0 {
				return true
			}
		}
	}
	return false
}

func areDiscoveredFieldsSameVirtualMachineStatus(s1, s2 cloudv1alpha1.VirtualMachineStatus) bool {
	if &s1 == &s2 {
		return true
	}
	if s1.Provider != s2.Provider {
		return false
	}
	if s1.Status != s2.Status {
		return false
	}
	if s1.VirtualPrivateCloud != s2.VirtualPrivateCloud {
		return false
	}
	if len(s1.Tags) != len(s2.Tags) ||
		len(s1.NetworkInterfaces) != len(s2.NetworkInterfaces) {
		return false
	}
	if !areTagsSame(s1.Tags, s2.Tags) {
		return false
	}
	if !areNetworkInterfaceReferencesSame(s1.NetworkInterfaces, s2.NetworkInterfaces) {
		return false
	}
	return true
}

func areDiscoveredFieldsSameNetworkInterfaceStatus(s1, s2 cloudv1alpha1.NetworkInterfaceStatus) bool {
	if &s1 == &s2 {
		return true
	}
	if strings.Compare(strings.ToLower(s1.MAC), strings.ToLower(s2.MAC)) != 0 {
		return false
	}
	if len(s1.Tags) != len(s2.Tags) ||
		len(s1.IPs) != len(s2.IPs) {
		return false
	}
	if !areTagsSame(s1.Tags, s2.Tags) {
		return false
	}
	if !areIPAddressesSame(s1.IPs, s2.IPs) {
		return false
	}
	return true
}

func areTagsSame(s1, s2 map[string]string) bool {
	for key1, value1 := range s1 {
		value2, found := s2[key1]
		if !found {
			return false
		}
		if strings.Compare(strings.ToLower(value1), strings.ToLower(value2)) != 0 {
			return false
		}
	}
	return true
}

func areNetworkInterfaceReferencesSame(s1, s2 []cloudv1alpha1.NetworkInterfaceReference) bool {
	if &s1 == &s2 {
		return true
	}

	s1NameMap := convertNetworkReferencesToMap(s1)
	s2NameMap := convertNetworkReferencesToMap(s2)
	for key1, value1 := range s1NameMap {
		value2, found := s2NameMap[key1]
		if !found {
			return false
		}
		if strings.Compare(strings.ToLower(value1.Name), strings.ToLower(value2.Name)) != 0 {
			return false
		}
		if strings.Compare(strings.ToLower(value1.Namespace), strings.ToLower(value2.Namespace)) != 0 {
			return false
		}
	}
	return true
}

func areIPAddressesSame(s1, s2 []cloudv1alpha1.IPAddress) bool {
	s1Map := convertAddressToMap(s1)
	s2Map := convertAddressToMap(s2)
	for key1 := range s1Map {
		_, found := s2Map[key1]
		if !found {
			return false
		}
	}
	return true
}

func convertAddressToMap(addresses []cloudv1alpha1.IPAddress) map[string]struct{} {
	ipAddressMap := make(map[string]struct{})
	for _, address := range addresses {
		key := fmt.Sprintf("%v:%v", address.AddressType, address.Address)
		ipAddressMap[key] = struct{}{}
	}
	return ipAddressMap
}

func convertNetworkReferencesToMap(nwRefs []cloudv1alpha1.NetworkInterfaceReference) map[string]cloudv1alpha1.NetworkInterfaceReference {
	nwRefMap := make(map[string]cloudv1alpha1.NetworkInterfaceReference)
	for _, ref := range nwRefs {
		nwRefMap[ref.Name] = ref
	}
	return nwRefMap
}

func updateCloudDiscoveredFieldsOfVirtualMachineStatus(current, discovered *cloudv1alpha1.VirtualMachineStatus) {
	current.Provider = discovered.Provider
	current.Status = discovered.Status
	current.NetworkInterfaces = discovered.NetworkInterfaces
	current.VirtualPrivateCloud = discovered.VirtualPrivateCloud
	current.Tags = discovered.Tags
}

func updateCloudDiscoveredFieldsOfNetworkInterfaceStatus(current, discovered *cloudv1alpha1.NetworkInterfaceStatus) {
	current.Tags = discovered.Tags
	current.MAC = discovered.MAC
	current.IPs = discovered.IPs
}

func updateAccountStatus(current, discovered *cloudv1alpha1.CloudProviderAccountStatus) {
	current.Error = discovered.Error
}
