package virtualmachinepolicy_test

import (
	"context"

	"antrea.io/cloudcontroller/apis/runtime/v1alpha1"
	. "antrea.io/cloudcontroller/pkg/apiserver/registry/virtualmachinepolicy"
	"antrea.io/cloudcontroller/pkg/controllers/cloud"
	logger "github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/client-go/tools/cache"
)

var (
	targetName = "targetname"
)

var _ = Describe("Virtualmachinepolicy", func() {
	var virtualMachinePolicyIndexer1 = cache.NewIndexer(
		func(obj interface{}) (string, error) {
			npStatus := obj.(*cloud.NetworkPolicyStatus)
			return npStatus.String(), nil
		},
		cache.Indexers{
			cloud.NetworkPolicyStatusIndexerByNamespace: func(obj interface{}) ([]string, error) {
				npStatus := obj.(*cloud.NetworkPolicyStatus)
				ret := []string{npStatus.Namespace}
				return ret, nil
			},
		})
	var virtualMachinePolicyIndexer2 = cache.NewIndexer(
		func(obj interface{}) (string, error) {
			npStatus := obj.(*cloud.NetworkPolicyStatus)
			return npStatus.String(), nil
		},
		cache.Indexers{
			cloud.NetworkPolicyStatusIndexerByNamespace: func(obj interface{}) ([]string, error) {
				npStatus := obj.(*cloud.NetworkPolicyStatus)
				ret := []string{npStatus.Namespace}
				return ret, nil
			},
		})
	var virtualMachinePolicyIndexer3 = cache.NewIndexer(
		func(obj interface{}) (string, error) {
			npStatus := obj.(*cloud.NetworkPolicyStatus)
			return npStatus.String(), nil
		},
		cache.Indexers{
			cloud.NetworkPolicyStatusIndexerByNamespace: func(obj interface{}) ([]string, error) {
				npStatus := obj.(*cloud.NetworkPolicyStatus)
				ret := []string{npStatus.Namespace}
				return ret, nil
			},
		})
	var l logger.Logger
	var npstatus1 = make(map[string]string)
	var npstatus2 = make(map[string]string)
	var npstatus3 = make(map[string]string)
	npstatus1["test1"] = "applied"
	npstatus2["test1"] = "applied"
	npstatus3["test1"] = "applied"
	npstatus2["test2"] = "in-progress"
	npstatus3["test2"] = "in-progress"
	npstatus3["test3"] = "error"
	cacheTest1 := &cloud.NetworkPolicyStatus{
		NamespacedName: types.NamespacedName{Namespace: "default", Name: "targetname"},
		NPStatus:       npstatus1,
	}
	cacheTest2 := &cloud.NetworkPolicyStatus{
		NamespacedName: types.NamespacedName{Namespace: "default", Name: "targetname"},
		NPStatus:       npstatus2,
	}
	cacheTest3 := &cloud.NetworkPolicyStatus{
		NamespacedName: types.NamespacedName{Namespace: "default", Name: "targetname"},
		NPStatus:       npstatus3,
	}
	expectedPolicy1 := &v1alpha1.VirtualMachinePolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "targetname",
		},
		Status: v1alpha1.VirtualMachinePolicyStatus{
			Realization: "SUCCESS",
			NetworkPolicyDetails: map[string]*v1alpha1.NetworkPolicyStatus{
				"test1": {
					Realization: "SUCCESS",
				},
			},
		},
	}
	expectedPolicy2 := &v1alpha1.VirtualMachinePolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "targetname",
		},
		Status: v1alpha1.VirtualMachinePolicyStatus{
			Realization: "IN-PROGRESS",
			NetworkPolicyDetails: map[string]*v1alpha1.NetworkPolicyStatus{
				"test1": {
					Realization: "SUCCESS",
				},
				"test2": {
					Realization: "IN-PROGRESS",
				},
			},
		},
	}
	expectedPolicy3 := &v1alpha1.VirtualMachinePolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "targetname",
		},
		Status: v1alpha1.VirtualMachinePolicyStatus{
			Realization: "FAILED",
			NetworkPolicyDetails: map[string]*v1alpha1.NetworkPolicyStatus{
				"test1": {
					Realization: "SUCCESS",
				},
				"test2": {
					Realization: "IN-PROGRESS",
				},
				"test3": {
					Realization: "FAILED",
					Reason:      "error",
				},
			},
		},
	}
	expectedPolicies := []*v1alpha1.VirtualMachinePolicy{
		expectedPolicy1,
		expectedPolicy2,
		expectedPolicy3,
	}
	Describe("Test Get function of Rest", func() {
		var npstatus1 = make(map[string]string)
		var npstatus2 = make(map[string]string)
		var npstatus3 = make(map[string]string)
		npstatus1["test1"] = "applied"
		npstatus2["test1"] = "applied"
		npstatus3["test1"] = "applied"
		npstatus2["test2"] = "in-progress"
		npstatus3["test2"] = "in-progress"
		npstatus3["test3"] = "error"
		_ = virtualMachinePolicyIndexer1.Update(cacheTest1)
		_ = virtualMachinePolicyIndexer2.Update(cacheTest2)
		_ = virtualMachinePolicyIndexer3.Update(cacheTest3)
		var virtualMachinePolicyIndexers = []cache.Indexer{virtualMachinePolicyIndexer1,
			virtualMachinePolicyIndexer2, virtualMachinePolicyIndexer3}
		It("Three status of realizing", func() {
			for i, virtualMachinePolicyIndexer := range virtualMachinePolicyIndexers {
				rest := NewREST(virtualMachinePolicyIndexer, l)
				actualGroupList, err := rest.Get(request.NewDefaultContext(), targetName, &metav1.GetOptions{})
				Expect(err).Should(BeNil())
				Expect(actualGroupList).To(Equal(expectedPolicies[i]))
			}

		})
	})
	Describe("Test List function of Rest", func() {
		expectedPoliyList := &v1alpha1.VirtualMachinePolicyList{
			Items: []v1alpha1.VirtualMachinePolicy{
				{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "targetname",
					},
					Status: v1alpha1.VirtualMachinePolicyStatus{
						Realization: "SUCCESS",
						NetworkPolicyDetails: map[string]*v1alpha1.NetworkPolicyStatus{
							"test1": {
								Realization: "SUCCESS",
							},
						},
					},
				},
			},
		}
		It("Should return the List result of Rest", func() {
			rest := NewREST(virtualMachinePolicyIndexer1, l)
			actualObj, err := rest.List(context.TODO(), &internalversion.ListOptions{})
			Expect(err).Should(BeNil())
			Expect(actualObj).To(Equal(expectedPoliyList))
		})
	})
})
