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
	"sync"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cloudv1alpha1 "antrea.io/antreacloud/apis/crd/v1alpha1"
	cloudprovider "antrea.io/antreacloud/pkg/cloud-provider"
	"antrea.io/antreacloud/pkg/cloud-provider/cloudapi/common"
)

const (
	externalEntitySelectorMatchIndexerByID  = "externalEntity.selector.id"
	externalEntitySelectorMatchIndexerByTag = "externalEntity.selector.tag"
	externalEntitySelectorMatchIndexerByVPC = "externalEntity.selector.vpc"
	virtualMachineIndexerByCloudAccount     = "virtualmachine.cloudaccount"
)

// CloudEntitySelectorReconciler reconciles a CloudEntitySelector object.
// nolint:golint
type CloudEntitySelectorReconciler struct {
	client.Client
	Log            logr.Logger
	Scheme         *runtime.Scheme
	CloudInventory *CloudInventory

	mutex      sync.Mutex
	accPollers map[types.NamespacedName]*accountPoller
}

// +kubebuilder:rbac:groups=crd.cloud.antrea.io,resources=cloudentityselectors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=crd.cloud.antrea.io,resources=cloudentityselectors/status,verbs=get;update;patch

func (r *CloudEntitySelectorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("cloudentityselector", req.NamespacedName)

	entitySelector := &cloudv1alpha1.CloudEntitySelector{}
	err := r.Get(ctx, req.NamespacedName, entitySelector)
	if err != nil && !errors.IsNotFound(err) {
		return ctrl.Result{}, err
	}
	if errors.IsNotFound(err) {
		err = r.processDelete(&req.NamespacedName)
		return ctrl.Result{}, err
	}

	err = r.processCreateOrUpdate(entitySelector, &req.NamespacedName)

	return ctrl.Result{}, err
}

func (r *CloudEntitySelectorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.accPollers = make(map[types.NamespacedName]*accountPoller)
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &cloudv1alpha1.VirtualMachine{},
		virtualMachineIndexerByCloudAccount, func(obj client.Object) []string {
			vm := obj.(*cloudv1alpha1.VirtualMachine)
			owner := vm.GetOwnerReferences()
			if len(owner) == 0 {
				return nil
			}
			return []string{owner[0].Name}
		}); err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&cloudv1alpha1.CloudEntitySelector{}).
		Complete(r)
}

func (r *CloudEntitySelectorReconciler) processCreateOrUpdate(selector *cloudv1alpha1.CloudEntitySelector,
	selectorNamespacedName *types.NamespacedName) error {
	accPoller, preExists := r.addAccountPoller(selector)

	if selector.Spec.VMSelector != nil {
		var vmMatches []interface{}
		for i := range selector.Spec.VMSelector.VMMatches {
			vmMatches = append(vmMatches, &selector.Spec.VMSelector.VMMatches[i])
		}
		if err := accPoller.vmMatches.Replace(vmMatches, ""); err != nil {
			r.Log.Error(err, "Unable to replace external entity selector matches", "Name", selector.Name)
		}

		vmList := &cloudv1alpha1.VirtualMachineList{}
		if err := r.List(context.TODO(), vmList, client.MatchingFields{
			virtualMachineIndexerByCloudAccount: selector.Name}, client.InNamespace(selector.Namespace)); err != nil {
			r.Log.Error(err, "Unable to get virtual machines for external entity selector", "Name", selector.Name)
		} else {
			var vms []*cloudv1alpha1.VirtualMachine
			for i := range vmList.Items {
				vm := &vmList.Items[i]
				// vm selector changed, trigger to recompute.
				vms = append(vms, vm)
			}
			if err := accPoller.doVirtualMachineOperations(vms); err != nil {
				r.Log.Error(err, "Unable to update virtual machines")
			}
		}
	}

	cloudInterface, err := cloudprovider.GetCloudInterface(common.ProviderType(accPoller.cloudType))
	if err != nil {
		if !preExists {
			_ = r.processDelete(selectorNamespacedName)
		}
		return err
	}

	err = cloudInterface.AddAccountResourceSelector(accPoller.namespacedName, selector)
	if err != nil {
		if !preExists {
			_ = r.processDelete(selectorNamespacedName)
		}
		r.Log.Info("selector add failed", "selector", selectorNamespacedName, "poller-exists", preExists)
		return err
	}

	if !preExists {
		go wait.Until(accPoller.doAccountPoller, time.Duration(accPoller.pollIntvInSeconds)*time.Second, accPoller.ch)
	}
	return nil
}

func (r *CloudEntitySelectorReconciler) processDelete(selectorNamespacedName *types.NamespacedName) error {
	poller := r.removeAccountPoller(selectorNamespacedName)
	if poller == nil {
		return nil
	}
	cloudInterface, err := cloudprovider.GetCloudInterface(common.ProviderType(poller.cloudType))
	if err != nil {
		return err
	}
	cloudInterface.RemoveAccountResourcesSelector(poller.namespacedName, selectorNamespacedName.Name)

	return nil
}

func (r *CloudEntitySelectorReconciler) addAccountPoller(selector *cloudv1alpha1.CloudEntitySelector) (*accountPoller, bool) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	selectorNamespacedName := &types.NamespacedName{
		Namespace: selector.Namespace,
		Name:      selector.Name,
	}

	if pollerScope, exists := r.accPollers[*selectorNamespacedName]; exists {
		r.Log.Info("poller exists", "selector", selectorNamespacedName)
		return pollerScope, exists
	}

	accountNamespacedName := &types.NamespacedName{
		Namespace: selector.Namespace,
		Name:      selector.Spec.AccountName,
	}
	account := &cloudv1alpha1.CloudProviderAccount{}
	_ = r.Get(context.TODO(), *accountNamespacedName, account)
	poller := &accountPoller{
		Client:            r.Client,
		scheme:            r.Scheme,
		log:               r.Log,
		pollIntvInSeconds: *account.Spec.PollIntervalInSeconds,
		cloudType:         account.Spec.ProviderType,
		namespacedName:    accountNamespacedName,
		selector:          selector.DeepCopy(),
		ch:                make(chan struct{}),
		cloudInventory:    r.CloudInventory,
	}

	poller.vmMatches = cache.NewIndexer(
		func(obj interface{}) (string, error) {
			_ = obj.(*cloudv1alpha1.VirtualMachineMatch)
			// The key is not important, re-write indexer each time cloud account is updated.
			poller.vmMatchKey++
			return fmt.Sprintf("%x", poller.vmMatchKey), nil
		},
		cache.Indexers{
			externalEntitySelectorMatchIndexerByID: func(obj interface{}) ([]string, error) {
				m := obj.(*cloudv1alpha1.VirtualMachineMatch)
				if m.VMMatch != nil && len(m.VMMatch.MatchID) > 0 {
					return []string{m.VMMatch.MatchID}, nil
				}
				return nil, nil
			},
			externalEntitySelectorMatchIndexerByTag: func(obj interface{}) ([]string, error) {
				m := obj.(*cloudv1alpha1.VirtualMachineMatch)
				if m.VMMatch == nil {
					return nil, nil
				}
				var tags []string
				for t := range m.VMMatch.MatchTags {
					tags = append(tags, t)
				}
				return tags, nil
			},
			externalEntitySelectorMatchIndexerByVPC: func(obj interface{}) ([]string, error) {
				m := obj.(*cloudv1alpha1.VirtualMachineMatch)
				if m.VpcMatch != nil && len(m.VpcMatch.MatchID) > 0 {
					return []string{m.VpcMatch.MatchID}, nil
				}
				return nil, nil
			},
		})
	r.accPollers[*selectorNamespacedName] = poller

	r.Log.Info("poller will be created", "selector", selectorNamespacedName)
	return poller, false
}

func (r *CloudEntitySelectorReconciler) removeAccountPoller(selectorNamespacedName *types.NamespacedName) *accountPoller {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	poller, found := r.accPollers[*selectorNamespacedName]
	if found {
		close(poller.ch)
		delete(r.accPollers, *selectorNamespacedName)
	}
	return poller
}
