package virtualmachinepolicy_test

import (
	"testing"

	"antrea.io/cloudcontroller/pkg/logging"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestVirtualmachinepolicy(t *testing.T) {
	logging.SetDebugLog(true)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Virtualmachinepolicy Suite")
}
