# Developers Guide

The project scaffold is created via [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder).
Please see these [instructions](https://book.kubebuilder.io/quick-start.html#installation) to
install  ``kubebuilder``.

If you are running Mac, consider using a Ubuntu 18 virtual machine.
(Recommended Configuration: Memory > 12288MB, Space > 100GB)

## Other Prerequisites

The following tools are required to build, test and run Antrea Plus Controllers.
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- [docker](https://docs.docker.com/install/)
- [kind](https://kind.sigs.k8s.io/docs/user/quick-start/)

## Build
Run the following commands to build AntreaPlus controllers.

```bash
make
```
The controller binaries can be located in ``./bin`` directory, and docker image ``antreaplus
/antreaplus:latest`` is created or updated in the local docker repository.  

## CRD Creation and Modification
The project current support single API group, all CRD Kind must be in a single API group 
``cloud.antreaplus.vmware-tanzu.com``. To create a new CRD, e.g. MyKind., do
```bash
kubebuilder create api --group cloud --version v1alpha1 --kind MyKind
```
The ``kubebuilder`` creates skeleton CRD definition in ``.api/cloud/v1alpha1/mykind_types.go``, and
a skeleton MyKindReconciler in ``./controllers/cloud/mykind_controller.go``. You will need to
1. Modify the new CRD definition as required. 
1. Move the skeleton reconciler file to appropriate location if needed.
1. Initiate the reconciler, if required, perhaps at process boot-up.

Then run 
```bash
make
make manifests
```
The first ``make`` attempts to build controller binaries, which in turn triggers code generation
based on the new CRD definition. The second ``make`` generates manifests for Roles with
permission to manipulate the new CRD.

## Enable Webhook

### Prerequisite
Webhook server requires [cert-manager](https://cert-manager.io/docs/installation/kubernetes/)
to provide SSL certificate support. It needs to installed to Antrea Plus K8s cluster.

```bash
kubectl create namespace cert-manager
kubectl apply --validate=false -f https://github.com/jetstack/cert-manager/releases/download/v1.2.0/cert-manager.yaml
```

### Enable webhook services
These are initial steps to enable webhook services on antrea-plus-cloud-controller. Skip to next section to just enable webhook for a CRD.
 
- In config/default/kustomization.yaml, uncomment all sections with ``CERTMANAGER`` and
 ``WEBHOOK``.
- In config/default/manager_webhook_patch.yaml, add Deployment patches to antrea-plus-cloud-controller. So that back-end Pods from
these Deployments becomes also webhook server.
- In config/webhook/service.yaml, add webhook services to antrea-plus-cloud-controller. 
- In config/certmanager/certificate.yaml. under dnsName, replace ``$(SERVICE_NAME)`` with
``*`` this allows the same Certificate be used by all webhook services in antrea-plus-system
namespace.  
- Create a yaml config/webhook/manifests-new.yaml, and create two pairs of
ValidatingWebhookConfiguration and MutatingWebhookConfiguration for each of the controllers, and
replace manifests.yaml with manifests-new.yaml in config/webhook/kustomization.yaml
-- In config/default/webhookcainjection_patch.yaml, add cainject patch to each of above
WebhookConfiguration.

### Enable webhook For CRD
- Choose which controller process that a CRD should run its webhook on. Run kubebuilder to generate
webhook skeleton implementation. The "ln" command allows webhook for CRD to be bootstrapped. In
the following example, we have decided VirtualMachine webhook shall be run on cloud
-controller, and VirtualMachine webhook implementation is under 
apis/crd/v1alpha1/virtualmachine_webhook.go.
```bash
ln -s cmd/cloud-controller/main.go main.go
kubebuilder create webhook --group cloud --version v1alpha1 --kind VirtualMachine --defaulting --programmatic-validation
rm main.go
```
- Auto-generate other implementation and K8s configuration manifests
```
make
make manifests
```
- In config/crd/kustomization.yaml, uncomment patches to the CRD.
- In config/crd/patches/webhook_in_CRD_NAME.yaml, add ``preserveUnknownFields: False`` under
 ``Spec``.
- In config/webhook/manifests-new.yaml, copy ClientCfg of CRD from config/webhook/manifests.yaml
and place it under webhookConfiguration of appropriate controller; And change 
``webhook.clientConfig.service.name`` to the webhook service of that controller.
- Run ``make manifests`` again to rebuild antrea-plus.yaml.

 
## TestBed and Deployment
The AntreaPlus manifest deploys one antrea-plus-cloud-controller Deployment. All
AntreaPlus resources are placed under Namespace antrea-plus-system  

### KinD
Create a KinD setup in the local machine.

```bash
./ci/kind/kind-setup.sh create kind
```
Get help for KinD setup by  
```bash
./ci/kind/kind-setup.sh help
````

If you do not have access to jfrog, tag and load AntreaPlus image manually
```bash
docker tag antreaplus/antreaplus vmwaresaas.jfrog.io/antrea-service-tilt-images/antreaplus/antreaplus
kind load docker-image vmwaresaas.jfrog.io/antrea-service-tilt-images/antreaplus/antreaplus
```
Apply manifest 
```bash
kubectl apply -f ./config/antrea-plus.yaml
``` 

### Cloud

See instructions to setup EKS/AKS [cluster](cloud.md)

Load AntreaPlus image and apply manifest to AKS
```bash
./terraform/aks load antreaplus/antreaplus:latest
./terraform/aks kubectl apply -f ./config/antrea-plus.yaml
```

Load AntreaPlus image and apply manifest to EKS
```bash
./terraform/eks load antreaplus/antreaplus:latest
./terraform/eks kubectl apply -f ./config/antrea-plus.yaml
```

## Unit Test
Unit tests are placed under same directories with the implementations, the test package name is
PACKAGE_test. For instance, unit test files xxx_test.go under pkg/cloud-provider have the package
name of cloudprovider_test.

Unit tests uses [go mock](https://github.com/golang/mock) to generate mock packages. To generate
mock code for a package, the package must implement interfaces, and add package/interfaces to
hack/mockgen.sh.
To generated mock mode, run 
```bash
make mock
```

To run unit tests,
```bash
make unit-test
```

## VM Agent test
VM Agent data path test may run as
```bash
make vm-agent-test
```

## Integration Test
Test/integration directory contains all integration tests. It uses
Ginkgo as the underlying frameworks. Each Ginkgo test spec may be run as default or extended
test in CI pipeline. The keywords `Core-test` and `Extended-test-*` are used in 
descriptions on any level of a test spec to indicate if this test spec should be run in zero 
or more test suites.

Set following variables to allow terraform to create AWS VPC, see [cloud](./cloud.md) for details
to install terraform.
```bash
export TF_VAR_aws_access_key_id=YOUR_AWS_KEY
export TF_VAR_aws_access_key_secret=YOUR_AWS_KEY_SECRET
export TF_VAR_aws_key_pair_name=YOU_AWS_KEY_PAIR
export TF_VAR_region=YOUR_AWS_REGION
export TF_VAR_owner=YOUR_ID
```
To run integration test,
```bash
ci/kind/kind-setup.sh create kind
make integration-test
```

you can also run integration tests on an existing K8s setup
```
make
kind load docker-image antreaplus/antreaplus
make integration-test
```

## Azure Agentless Integration Test
Set following variables to allow terraform to create Azure VNET.
```bash
export TF_VAR_azure_client_id=YOUR_AZURE_CLIENT_ID
export TF_VAR_azure_client_subscription_id=YOUR_AZURE_CLIENT_SUBSCRIPTION_ID
export TF_VAR_azure_client_secret=YOUR_AZURE_CLIENT_SECRET
export TF_VAR_azure_client_tenant_id=YOUR_AZURE_TENANT_ID
```
To run integration test,
```bash
ci/kind/kind-setup.sh create kind
make azure-agentless-integration-test
```

you can also run integration tests on an existing K8s setup
```
make
kind load docker-image antreaplus/antreaplus
make azure-agentless-integration-test
```

## Federation Integration Test
The federation integration test make use of 2 KinD cluster. Currently it is hardcoded to kind
-kind and kind-kind1. So you must already have created these two clusters. i.e

```bash
ci/kind/kind-setup.sh create kind
ci/kind/kind-setup.sh create kind1
make federation-test
```

## Agent Integration Test

### In EKS
Integration test may run in agented mode with an EKS cluster. You will need to have an EKS
cluster already created as described in [cloud](./cloud.md), in 
addition to integration test requirements.

The following pushes docker image to your account in dockerhub, and notify terraform the your
 docker account credential.
```bash
docker tag antreaplus/vmagent YOUR_DOCKER_ID/vmagent
docker push YOUR_DOCKER_ID/vmagent
    
export TF_VAR_aws_docker_usr=YOUR_DOCKER_ID
export TF_VAR_aws_docker_pwd=YOUR_DOCKER_PWD
```

To run EKS agented test,
```bash
~/terraform/eks load antreaplus/antreaplus
export AGENT_KUBE_CONFIG=path_to_eks_kubeconfig
make vm-agent-eks-integration-test
```

### In AKS
Integration test may run in agented mode with an AKS cluster. You will need to have an AKS
cluster already created as described in [cloud](./cloud.md), in addition to integration test
requirements.
```bash
docker tag antreaplus/vmagent YOUR_DOCKER_ID/vmagent
docker push YOUR_DOCKER_ID/vmagent
export TF_VAR_azure_docker_usr=YOUR_DOCKER_ID
export TF_VAR_azure_docker_pwd=YOUR_DOCKER_PWD

~/terraform/eks load antreaplus/antreaplus
export AGENT_KUBE_CONFIG=path_to_aks_kubeconfig
make vm-agent-aks-integration-test
```

## Debug Containers

In a KinD cluster, we can attach debug running containers as described
 [here](https://kubernetes.io/docs/tasks/debug-application-cluster/debug-running-pod/).

you must used kubectl version 1.18 or later. For instance, the folowing command attaches a
busybox into antrea-plus-cloud-controllers namespace.
 ```bash
kubectl debug -it antrea-plus-cloud-controller-59b4fc6bbf-kdqd2  --image=busybox -n antrea-plus-system --target=antrea-plus-cloud-controller
Defaulting debug container name to debugger-qdkwv.
If you don't see a command prompt, try pressing enter.
/ #
```
