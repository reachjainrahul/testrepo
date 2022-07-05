/*
 *  Copyright  (c) 2020 VMWare, Inc.  All rights reserved. -- VMWare Confidential
 */

package networkinterface

import (
	controlplane "antrea.io/antreacloud/apis/controlplane/v1alpha1"
	controllers "antrea.io/antreacloud/pkg/controllers/cloud"
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/api/meta"
	metatable "k8s.io/apimachinery/pkg/api/meta/table"
	"k8s.io/apiserver/pkg/endpoints/request"

	"antrea.io/antreacloud/apis/controlplane/v1alpha1"
	logger "github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/registry/rest"
)

// REST implements rest.Storage for Network Interface.
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
	return &v1alpha1.NetworkInterface{}
}

func (r *REST) NewList() runtime.Object {
	return &v1alpha1.NetworkInterfaceList{}
}

func (r *REST) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	ns, ok := request.NamespaceFrom(ctx)
	if !ok || len(ns) == 0 {
		return nil, errors.NewBadRequest("Namespace parameter required.")
	}
	nic, exists := r.inventory.GetNetworkInterface(ns, name)
	if !exists {
		return nil, errors.NewNotFound(schema.GroupResource{
			Group:    options.ResourceVersion,
			Resource: options.ResourceVersion}, name)
	}
	return nic, nil
}

func (r *REST) List(ctx context.Context, options *internalversion.ListOptions) (runtime.Object, error) {
	ns, _ := request.NamespaceFrom(ctx)
	nics := r.inventory.ListNetworkInterfaces(ns)
	nicList := &controlplane.NetworkInterfaceList{
		Items: nics,
	}
	return nicList, nil
}

func (r *REST) NamespaceScoped() bool {
	return true
}

func (r *REST) ConvertToTable(ctx context.Context, obj runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
	table := &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{Name: "Name", Type: "string", Description: "Network Interface Name."},
			{Name: "Owner-Id", Type: "string", Description: "Owner ID of the Network Interface."},
			{Name: "Owner-Type", Type: "string", Description: "Owner Type of the Network Interface."},
			{Name: "Internal-IP", Type: "string", Description: "Private IP of the Network Interface."},
			{Name: "External-IP", Type: "string", Description: "Public IP of the Network Interface."},
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
			nic := obj.(*controlplane.NetworkInterface)
			var privateIP, publicIP string
			for _, ip := range nic.Status.IPs {
				if ip.AddressType == controlplane.AddressTypeInternalIP {
					privateIP = ip.Address
				} else {
					publicIP = ip.Address
				}
			}
			if len(nic.OwnerReferences) == 0 {
				r.logger.Error(fmt.Errorf("owner cannot be zero"), "For NIC", "nic", nic.Name)
			}
			return []interface{}{name, nic.OwnerReferences[0].Name,
				nic.OwnerReferences[0].Kind, privateIP, publicIP}, nil
		})
	return table, err
}
