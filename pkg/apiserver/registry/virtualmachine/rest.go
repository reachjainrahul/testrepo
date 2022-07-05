/*
 *  Copyright  (c) 2020 VMWare, Inc.  All rights reserved. -- VMWare Confidential
 */

package virtualmachine

import (
	controlplane "antrea.io/antreacloud/apis/controlplane/v1alpha1"
	controllers "antrea.io/antreacloud/pkg/controllers/cloud"
	"context"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apiserver/pkg/endpoints/request"

	logger "github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	metatable "k8s.io/apimachinery/pkg/api/meta/table"
	"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/registry/rest"
)

// REST implements rest.Storage for VirtualMachine.
type REST struct {
	inventory *controllers.CloudInventory
	logger    logger.Logger
}

var (
	_ rest.Scoper = &REST{}
	_ rest.Getter = &REST{}
	_ rest.Lister = &REST{}
)

// NewREST returns a REST object that will work against API services.
func NewREST(inventory *controllers.CloudInventory, l logger.Logger) *REST {
	return &REST{
		inventory: inventory,
		logger:    l}
}

func (r *REST) New() runtime.Object {
	return &controlplane.VirtualMachine{}
}

func (r *REST) NewList() runtime.Object {
	return &controlplane.VirtualMachineList{}
}

func (r *REST) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	ns, ok := request.NamespaceFrom(ctx)
	if !ok || len(ns) == 0 {
		return nil, errors.NewBadRequest("Namespace parameter required.")
	}
	vm, exists := r.inventory.GetVirtualMachine(ns, name)
	if !exists {
		return nil, errors.NewNotFound(schema.GroupResource{
			Group:    options.ResourceVersion,
			Resource: options.ResourceVersion}, name)
	}
	return vm, nil
}

func (r *REST) List(ctx context.Context, options *internalversion.ListOptions) (runtime.Object, error) {
	ns, _ := request.NamespaceFrom(ctx)
	vms := r.inventory.ListVirtualMachines(ns)
	vmList := &controlplane.VirtualMachineList{
		Items: vms,
	}
	return vmList, nil
}

func (r *REST) NamespaceScoped() bool {
	return true
}

func (r *REST) ConvertToTable(ctx context.Context, obj runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
	table := &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{Name: "Name", Type: "string", Description: "Virtual Machine Name."},
			{Name: "Cloud-Provider", Type: "string", Description: "Cloud Provider of the VM."},
			{Name: "Virtual-Private-Cloud", Type: "string", Description: "Virtual Private Cloud of the VM."},
			{Name: "Status", Type: "string", Description: "Current state of the VM"},
		},
	}
	if m, err := meta.ListAccessor(obj); err == nil {
		table.ResourceVersion = m.GetResourceVersion()
		table.Continue = m.GetContinue()
		table.RemainingItemCount = m.GetRemainingItemCount()
	} else {
		if m, err := meta.CommonAccessor(obj); err == nil {
			table.ResourceVersion = m.GetResourceVersion()
		}
	}
	var err error
	table.Rows, err = metatable.MetaToTableRow(obj,
		func(obj runtime.Object, m metav1.Object, name, age string) ([]interface{}, error) {
			vm := obj.(*controlplane.VirtualMachine)
			return []interface{}{name, vm.Status.Provider, vm.Status.VirtualPrivateCloud, vm.Status.Status}, nil
		})
	return table, err
}
