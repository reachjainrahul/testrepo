/*******************************************************************************
 * Copyright 2022 VMWare, Inc.  All rights reserved. -- VMWare Confidential
 *******************************************************************************/

package cloud

import (
	controlplane "antrea.io/antreacloud/apis/controlplane/v1alpha1"
	crd "antrea.io/antreacloud/apis/crd/v1alpha1"
	"antrea.io/antreacloud/pkg/cloud-provider/cloudapi/common"
	logger "github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
)

const (
	VirtualMachineIndexerByVPCID       = "metadata.annotations.cloud-assigned-vpc-id"
	VirtualMachineIndexerByNamespace   = "metadata.namespace"
	NetworkInterfaceIndexerByVPCID     = "metadata.annotations.cloud-assigned-vpc-id"
	NetworkInterfaceIndexerByNamespace = "metadata.namespace"
)

type CloudInventory struct {
	logger       logger.Logger
	vmInventory  cache.Indexer
	nicInventory cache.Indexer
}

func NewCloudInventory(logger logger.Logger) *CloudInventory {
	cloudInventory := &CloudInventory{
		logger: logger,
	}
	cloudInventory.vmInventory = cache.NewIndexer(
		func(obj interface{}) (string, error) {
			vm := obj.(*controlplane.VirtualMachine)
			// vm.Name is cloud vm instance-ID
			return types.NamespacedName{Name: vm.Name, Namespace: vm.Namespace}.String(), nil
		},
		cache.Indexers{
			// TODO: Account ID field is not present. Extend Virtual Machine type to add Account ID
			VirtualMachineIndexerByVPCID: func(obj interface{}) ([]string, error) {
				vm := obj.(*controlplane.VirtualMachine)
				cloudID := vm.Annotations[common.AnnotationCloudAssignedVPCIDKey]
				return []string{cloudID}, nil
			},
			VirtualMachineIndexerByNamespace: func(obj interface{}) ([]string, error) {
				vm := obj.(*controlplane.VirtualMachine)
				return []string{vm.Namespace}, nil
			},
		})
	cloudInventory.nicInventory = cache.NewIndexer(
		func(obj interface{}) (string, error) {
			nic := obj.(*controlplane.NetworkInterface)
			// nic.Name is cloud nic cloud ID
			return types.NamespacedName{Name: nic.Name, Namespace: nic.Namespace}.String(), nil
		},
		cache.Indexers{
			// TODO: Evaluate if we need Network Interface @archana
			NetworkInterfaceIndexerByVPCID: func(obj interface{}) ([]string, error) {
				nic := obj.(*controlplane.NetworkInterface)
				cloudID := nic.Annotations[common.AnnotationCloudAssignedVPCIDKey]
				return []string{cloudID}, nil
			},
			NetworkInterfaceIndexerByNamespace: func(obj interface{}) ([]string, error) {
				nic := obj.(*controlplane.NetworkInterface)
				return []string{nic.Namespace}, nil
			},
		})
	return cloudInventory
}

// ListVirtualMachines list virtual machine per namespace or all
func (i *CloudInventory) ListVirtualMachines(ns string) []controlplane.VirtualMachine {
	var vms []interface{}
	if len(ns) == 0 {
		vms = i.vmInventory.List()
	} else {
		vms, _ = i.vmInventory.ByIndex(VirtualMachineIndexerByNamespace, ns)
	}
	vmList := make([]controlplane.VirtualMachine, len(vms))
	for i, vm := range vms {
		vmList[i] = *(vm.(*controlplane.VirtualMachine))
	}
	return vmList
}

// GetVirtualMachine Returns Virtual Machine for given namespace and name
func (i *CloudInventory) GetVirtualMachine(ns, name string) (*controlplane.VirtualMachine, bool) {
	vmKey := types.NamespacedName{Name: name, Namespace: ns}.String()
	if vm, ok, _ := i.vmInventory.GetByKey(vmKey); !ok {
		return nil, false
	} else {
		return vm.(*controlplane.VirtualMachine), true
	}
}

// CopyVm Copy from CRD VM to ControlPlane VM. It should be removed
func (i *CloudInventory) copyVm(vm *crd.VirtualMachine) *controlplane.VirtualMachine {
	newVM := controlplane.VirtualMachine{
		ObjectMeta: vm.ObjectMeta,
		TypeMeta:   vm.TypeMeta,
	}
	// Copy Field by Field
	newVM.Status.Provider = vm.Status.Provider
	newVM.Status.Status = vm.Status.Status
	newVM.Status.VirtualPrivateCloud = vm.Status.VirtualPrivateCloud
	return &newVM
}

// AddVirtualMachine Add Virtual Machine to Inventory
func (i *CloudInventory) AddVirtualMachine(vm *crd.VirtualMachine) {
	i.logger.Info("Adding Virtual Machine.", "Namespace", vm.Namespace, "Name", vm.Name)
	vmNew := i.copyVm(vm) // TODO: This Copy will go away
	if err := i.vmInventory.Add(vmNew); err != nil {
		i.logger.Error(err, "Failed to add Virtual Machine", "vm", vm.Name)
	}
}

// DeleteVirtualMachine Delete Virtual Machine from Inventory
func (i *CloudInventory) DeleteVirtualMachine(vm *crd.VirtualMachine) {
	i.logger.Info("Deleting Virtual Machine.", "Namespace", vm.Namespace, "Name", vm.Name)
	vmNew := i.copyVm(vm)
	if err := i.vmInventory.Delete(vmNew); err != nil {
		i.logger.Error(err, "Failed to delete Virtual Machine", "vm", vm.Name)
	}
}

// UpdateVirtualMachine Update Virtual Machine from Inventory
func (i *CloudInventory) UpdateVirtualMachine(vm *crd.VirtualMachine) {
	i.logger.Info("Updating Virtual Machine.", "Namespace", vm.Namespace, "Name", vm.Name)
	vmNew := i.copyVm(vm)
	if err := i.vmInventory.Update(vmNew); err != nil {
		i.logger.Error(err, "Failed to Update Virtual Machine", "vm", vm.Name)
	}
}

// ListNetworkInterfaces list network interfaces per namespace or all
func (i *CloudInventory) ListNetworkInterfaces(ns string) []controlplane.NetworkInterface {
	var nics []interface{}
	if len(ns) == 0 {
		nics = i.nicInventory.List()
	} else {
		nics, _ = i.nicInventory.ByIndex(NetworkInterfaceIndexerByNamespace, ns)
	}
	nicList := make([]controlplane.NetworkInterface, len(nics))
	for i, nic := range nics {
		nicList[i] = *(nic.(*controlplane.NetworkInterface))
	}
	return nicList
}

// GetNetworkInterface Returns Network Interface for given namespace and name
func (i *CloudInventory) GetNetworkInterface(ns, name string) (*controlplane.NetworkInterface, bool) {
	nicKey := types.NamespacedName{Name: name, Namespace: ns}.String()
	if nic, ok, _ := i.nicInventory.GetByKey(nicKey); !ok {
		return nil, false
	} else {
		return nic.(*controlplane.NetworkInterface), true
	}
}

//TODO: Merge Network interface objects in VM CRD, disabling Network interface related code for now.
/*
// CopyNic Copy from CRD Nic to ControlPlane Nic. It should be removed
func (i *CloudInventory) copyNic(nic *crd.NetworkInterface) *controlplane.NetworkInterface {
	newNic := controlplane.NetworkInterface{
		ObjectMeta: nic.ObjectMeta,
		TypeMeta:   nic.TypeMeta,
	}
	// Copy Field by Field
	newNic.Status.Tags = nic.Status.Tags
	newNic.Status.MAC = nic.Status.MAC
	for _, ip := range nic.Status.IPs {
		tempIP := controlplane.IPAddress{}
		tempIP.Address = ip.Address
		tempIP.AddressType = controlplane.AddressType(ip.AddressType)
		newNic.Status.IPs = append(newNic.Status.IPs, tempIP)
	}
	return &newNic
}

// AddNetworkInterface Add Network Interface to Inventory
func (i *CloudInventory) AddNetworkInterface(nic *crd.NetworkInterface) {
	i.logger.Info("Adding Network Interface.", "Namespace", nic.Namespace, "Name", nic.Name)
	nicNew := i.copyNic(nic) // TODO: This Copy will go away
	if err := i.nicInventory.Add(nicNew); err != nil {
		i.logger.Error(err, "Failed to add Network Interface", "nic", nic.Name)
	}
}

// DeleteNetworkInterface Delete Network Interface from Inventory
func (i *CloudInventory) DeleteNetworkInterface(nic *crd.NetworkInterface) {
	i.logger.Info("Deleting Network Interface.", "Namespace", nic.Namespace, "Name", nic.Name)
	nicNew := i.copyNic(nic)
	if err := i.nicInventory.Delete(nicNew); err != nil {
		i.logger.Error(err, "Failed to delete Network Interface", "nic", nic.Name)
	}
}

// UpdateNetworkInterface Update Network Interface from Inventory
func (i *CloudInventory) UpdateNetworkInterface(nic *crd.NetworkInterface) {
	i.logger.Info("Updating Network Interface.", "Namespace", nic.Namespace, "Name", nic.Name)
	nicNew := i.copyNic(nic)
	if err := i.nicInventory.Update(nicNew); err != nil {
		i.logger.Error(err, "Failed to Update Network Interface", "vm", nic.Name)
	}
}
*/
