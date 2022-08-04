# Developers Guide

The project scaffold is created via [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder).
To install `kubebuilder version 3`, please refer to
[kubebuilder quick start guide](https://book.kubebuilder.io/quick-start.html#installation).

## Prerequisites

The following tools are required to build, test and run Cloud Controller
`cloud-controller`.

- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- [docker](https://docs.docker.com/install/)
- [kind](https://kind.sigs.k8s.io/docs/user/quick-start/)

## Build

Build Cloud Controller `cloud-controller` image. The `cloud-controller` binary is
located in `./bin` directory and the docker image `antrea/cloud-controller:latest`
is created or updated in the local docker repository.

```bash
$ make
```

## CRD Creation and Modification

Cloud Controller currently has a single API group `crd.cloud.antrea.io`. This
section specifies the steps required to create a new CRD. `MyKind` is used an
example CRD object for illustration.

1. Kubebuilder expects `main.go` in the project home directory. So before
   running any kubebuilder command, create a soft link for `main.go` in project
   home directory.

```bash
$ ln -s cmd/cloud-controller/main.go main.go
```

2. Run kubebuilder create API command, to create a new API group for the kind
   object. This command initiates a reconciler in `main.go`.

```bash
$ kubebuilder create api --group cloud --version v1alpha1 --kind MyKind
```

The kubebuilder creates a skeleton CRD definition in
`apis/cloud/v1alpha1/mykind_types.go`, and a skeleton `MyKindReconciler` in
`controllers/cloud/mykind_controller.go`.

3. Modify the new CRD definition as required.
4. Move skeleton reconciler file `controllers/cloud/mykind_controller.go` under
   `pkg/controller/cloud` and remove`./controllers` directory.
5. Remove the import path of skeleton reconciler `controller/cloud` from
   `main.go`, since skeleton reconciler file is moved under
   `pkg/controller/cloud`.
6. Remove `main.go` from project home directory.

```bash
$ rm main.go
```

7. Then run following make commands.

```bash
$ make
$ make manifests
```

The first make command builds the controller binary, which in turn triggers code
generation based on the new CRD definition. The second make command generates a
manifests for Roles with permission to manipulate the new CRD.

### Enable Webhook For CRD

To create a webhook for the newly created CRD, follow the below procedure:

1. Make sure that `main.go` exists in project home directory. Run kubebuilder
   command to create a webhook skeleton implementation and a webhook for the CRD
   is bootstrapped in `main.go`.

```bash
$ ln -s cmd/cloud-controller/main.go main.go
$ kubebuilder create webhook --group cloud --version v1alpha1 --kind MyKind --defaulting --programmatic-validation
$ rm main.go
```

2. Auto-generate other required implementations and K8s configuration manifests.

```bash
$ make
$ make manifests
```

3. In `config/crd/kustomization.yaml`, uncomment patches to the CRD.
4. In `config/crd/patches/webhook_in_CRD_NAME.yaml`, add
   `preserveUnknownFields: False` under the Spec. Also change
   `webhook.clientConfig.service.name` to the webhook service of the controller.
5. In `config/webhook/manifests-new.yaml`, copy the ClientCfg of CRD from
   `config/webhook/manifests.yaml` and place it under webhook configuration of
   `cloud controller`. Change `webhook.clientConfig.service.name` to the webhook
   service of the controller.
6. Run `make manifests` again to rebuild `cloud-controller.yaml`.

## TestBed and Deployment

The Cloud Controller manifest deploys one `cloud-controller` Deployment.
All the Cloud Controller related resources are namespaced under `kube-system`
.

```bash
$ kubectl get deployment -A
NAMESPACE            NAME                      READY   UP-TO-DATE   AVAILABLE   AGE
cert-manager         cert-manager              1/1     1            1           41m
cert-manager         cert-manager-cainjector   1/1     1            1           41m
cert-manager         cert-manager-webhook      1/1     1            1           41m
kube-system          antrea-controller         1/1     1            1           41m
kube-system          cloud-controller          1/1     1            1           40m
kube-system          coredns                   2/2     2            2           43m
local-path-storage   local-path-provisioner    1/1     1            1           43m

$ kubectl get pods -A 
NAMESPACE            NAME                                         READY   STATUS    RESTARTS   AGE
cert-manager         cert-manager-677874db78-mxp8g                1/1     Running   0          42m
cert-manager         cert-manager-cainjector-6c5bf7b759-spn9w     1/1     Running   0          42m
cert-manager         cert-manager-webhook-5685fdbc4b-kdrdn        1/1     Running   0          42m
kube-system          antrea-agent-6rrbn                           2/2     Running   0          42m
kube-system          antrea-agent-6szv8                           2/2     Running   0          42m
kube-system          antrea-agent-7ggx9                           2/2     Running   0          42m
kube-system          antrea-controller-5f9bfb6b5d-p8znw           1/1     Running   0          37m
kube-system          cloud-controller-7f4795f64b-6hbn5            1/1     Running   0          33m
kube-system          coredns-5598b8945f-6j7lp                     1/1     Running   0          43m
kube-system          coredns-5598b8945f-qggsg                     1/1     Running   0          43m
kube-system          etcd-kind-control-plane                      1/1     Running   0          44m
kube-system          kube-apiserver-kind-control-plane            1/1     Running   0          44m
kube-system          kube-controller-manager-kind-control-plane   1/1     Running   0          44m
kube-system          kube-proxy-d7ssn                             1/1     Running   0          43m
kube-system          kube-proxy-hrfm5                             1/1     Running   0          43m
kube-system          kube-proxy-rfzmn                             1/1     Running   0          43m
kube-system          kube-scheduler-kind-control-plane            1/1     Running   0          44m
local-path-storage   local-path-provisioner-5ddd94ff66-gdqzg      1/1     Running   0          43m
```

### Kind based cluster deployment

Create a Kind setup in the local machine.

```bash
$ ./ci/kind/kind-setup.sh create kind
```

Get help for Kind setup.

```bash
$ ./ci/kind/kind-setup.sh help
````

Apply manifest.

```bash
$ kubectl apply -f ./config/cloud-controller.yml
``` 

### Cloud based cluster deployment

Please refer [EKS](eks-installation.md) / [AKS](aks-installation.md) guide for
cloud based cluster deployment.

Load Cloud Controller image and apply manifest to AKS cluster.

```bash
$ ./terraform/aks load antrea/cloud-controller:latest
$ ./terraform/aks kubectl apply -f ./config/cloud-controller.yml
```

Load Cloud Controller image and apply manifest to EKS cluster.

```bash
$ ./terraform/eks load antrea/cloud-controller:latest
$ ./terraform/eks kubectl apply -f ./config/cloud-controller.yml
```

## Unit Test

Unit tests are placed under same directories with the implementations. The test
package name is `PACKAGE_test`. For instance, the unit test files `xxx_test.go`
under `pkg/cloud-provider` have the package name of `cloudprovider_test`. Unit
tests uses [go mock](https://github.com/golang/mock) to generate mock packages.
To generate mock code for a package, the package must implement interfaces, and
add package/interfaces to [mockgen](../hack/mockgen.sh).

To generate mock:

```bash
$ make mock
```

To run unit tests:

```bash
$ make unit-test
```

## Integration Test

The `test/integration` directory contains all the integration tests. It uses
Ginkgo as the underlying framework. Each Ginkgo test spec may be run as default
or extended test in the CI pipeline. The keywords `Core-test` and
`Extended-test-*` are used in descriptions on any level of a test spec, to
indicate if this test spec should be run in zero or more test suites.

The test creates a VPC in AWS and deploys 3 VMs. Sets the following variables to
allow terraform to create an AWS VPC. Please refer to [eks guide](eks-installation.md)
for the deployment details.

```bash
$ export TF_VAR_aws_access_key_id=YOUR_AWS_KEY
$ export TF_VAR_aws_access_key_secret=YOUR_AWS_KEY_SECRET
$ export TF_VAR_aws_key_pair_name=YOU_AWS_KEY_PAIR
$ export TF_VAR_region=YOUR_AWS_REGION
$ export TF_VAR_owner=YOUR_ID
```

To run integration test,

```bash
$ ci/kind/kind-setup.sh create kind
$ make integration-test
```

You can also run integration tests on an existing K8s setup.

```bash
$ make
$ kind load docker-image antrea/cloud-controller
$ make integration-test
```

## Azure Integration Test

Set the following variables to allow terraform scripts to create an Azure VNET
and a compute VNET with 3 VMs. Please refer to [aks guide](aks-installation.md)
for the deployment details.

```bash
$ export TF_VAR_azure_client_id=YOUR_AZURE_CLIENT_ID
$ export TF_VAR_azure_client_subscription_id=YOUR_AZURE_CLIENT_SUBSCRIPTION_ID
$ export TF_VAR_azure_client_secret=YOUR_AZURE_CLIENT_SECRET
$ export TF_VAR_azure_client_tenant_id=YOUR_AZURE_TENANT_ID
```

To run integration test:

```bash
$ ci/kind/kind-setup.sh create kind
$ make azure-integration-test
```

You can also run integration tests on an existing K8s setup:

```bash
$ make
$ kind load docker-image antrea/cloud-controller
$ make azure-integration-test
```
