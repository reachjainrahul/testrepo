# Users Guide

## Prerequisites

* [kubectl](https://kubernetes.io/docs/tasks/tools/) installed.
* An active Kubernetes cluster, accessible using kubectl.
* [Antrea](https://github.com/antrea-io/antrea/) deployed. Recommend v1.7.
* [cert-manager](https://github.com/jetstack/cert-manager) deployed. Recommend
  v1.8.

## Installation

### Deploying Nephe in a Kind cluster

Create a Kind Cluster. Recommend Kind v0.12.

```bash
$ ./ci/kind/kind-setup.sh create kind
```

Install Nephe.

```bash
$ kubectl apply -f config/nephe.yml
```

### Deploying Nephe in EKS cluster

To deploy Nephe on an EKS cluster, please refer
to [the EKS installation guide](eks-installation.md).

### Deploying Nephe in AKS cluster

To deploy Nephe on an AKS cluster, please refer
to [the AKS installation guide](aks-installation.md).

## Importing Cloud VMs

To manage security policies of VMs, we need to first import target VMs onto the
`nephe-controller`. Below sections sets up access to public cloud account,
select target VMs, and import VMs into the K8s cluster as `VirtualMachine` CRs.

### CloudProviderAccount

To import cloud VMs, user needs to configure a `CloudProviderAccount` CR, with
the cloud account credentials.

* Sample `CloudProviderAccount` for AWS:

```bash
$ kubectl create namespace sample-ns
$ cat <<EOF | kubectl apply -f -
apiVersion: crd.cloud.antrea.io/v1alpha1
kind: CloudProviderAccount
metadata:
  name: cloudprovideraccount-sample
  namespace: sample-ns
spec:
  awsConfig:
    accountID: "<REPLACE_ME>"
    accessKeyId: "<REPLACE_ME>"
    accessKeySecret: "<REPLACE_ME>"
    region: "<REPLACE_ME>"
EOF
``` 

* Sample `CloudProviderAccount` for Azure:

```bash
$ kubectl create namespace sample-ns
$ cat <<EOF | kubectl apply -f -
apiVersion: crd.cloud.antrea.io/v1alpha1
kind: CloudProviderAccount
metadata:
  name: cloudprovideraccount-sample
  namespace: sample-ns
spec:
  azureConfig:
    subscriptionId: "<REPLACE_ME>"
    clientId: "<REPLACE_ME>"
    tenantId: "<REPLACE_ME>"
    clientKey: "<REPLACE_ME>"
    region: "<REPLACE_ME>"
EOF
``` 

### CloudEntitySelector

Once a `CloudProviderAccount` CR is added, virtual machines (VMs) may be
imported in the same Namespace via `CloudEntitySelector` CRD. The below example
selects VMs in VPC `VPC_ID` from `cloudprovideraccount-sample` to import in
`sample-ns` Namespace.

```bash
$ cat <<EOF | kubectl apply -f -
apiVersion: crd.cloud.antrea.io/v1alpha1
kind: CloudEntitySelector
metadata:
  name: cloudentityselector-sample01
  namespace: sample-ns
spec:
  accountName: cloudprovideraccount-sample
  vmSelector:
      - vpcMatch:
          matchID: "<VPC_ID>"
EOF
``` 

If there are any virtual machines in VPC `VPC_ID`, those virtual machines will
be imported. Invoke kubectl commands to get the details of imported VMs.

```bash
$ kubectl get virtualmachines -A
$ kubectl get vm -A
NAMESPACE        NAME                  CLOUD-PROVIDER   VIRTUAL-PRIVATE-CLOUD   STATUS
sample-ns        i-01b09fee2f216c1d7   AWS              vpc-02d3e1e0f15a56f4b   running
sample-ns        i-02a0b61c39cb34e5c   AWS              vpc-02d3e1e0f15a56f4b   running
sample-ns        i-0ae693c487e22dca8   AWS              vpc-02d3e1e0f15a56f4b   running
```

Currently, the following matching criteria are supported to import VMs.

* AWS:
    * vpcMatch: matchID, matchName
    * vmMatch: matchID, matchName
* Azure:
    * vpcMatch: matchID
    * vmMatch: matchID, matchName

### External Entity

For each cloud VM, an `ExternalEntity` CR is created, which can be used to
configure AntreaNetworkPolicy(ANP).

```bash
$ kubectl get externalentities -A
$ kubectl get ee -A
NAMESPACE   NAME                                 AGE
sample-ns   virtualmachine-i-05331c205bc6df47f   2m9s
sample-ns   virtualmachine-i-072a347128237cc63   2m9s
sample-ns   virtualmachine-i-08c3eb2ada5f85e02   2m9s
```

```bash
$ kubectl describe ee virtualmachine-i-05331c205bc6df47f -n sample-ns
Name:         virtualmachine-i-05331c205bc6df47f
Namespace:    sample-ns
Labels:       environment.tag.nephe=nephe
              kind.nephe=virtualmachine
              login.tag.nephe=ubuntu
              name.nephe=i-05331c205bc6df47f
              name.tag.nephe=vpc-0cfddb48a8119837e-ubuntu1
              namespace.nephe=sample-ns
              svcusercloudantreaio.tag.nephe=vm-http8080
              terraform.tag.nephe=true
              vpc.nephe=vpc-0cfddb48a8119837e
Annotations:  <none>
API Version:  crd.antrea.io/v1alpha2
Kind:         ExternalEntity
Metadata:
  Creation Timestamp:  2022-07-20T20:57:42Z
  Generation:          1
  Managed Fields:
    API Version:  crd.antrea.io/v1alpha2
    Fields Type:  FieldsV1
    fieldsV1:
      f:metadata:
        f:labels:
          .:
          f:environment.tag.nephe:
          f:kind.nephe:
          f:login.tag.nephe:
          f:name.nephe:
          f:name.tag.nephe:
          f:namespace.nephe:
          f:svcusercloudantreaio.tag.nephe:
          f:terraform.tag.nephe:
          f:vpc.nephe:
        f:ownerReferences:
      f:spec:
        .:
        f:endpoints:
        f:externalNode:
    Manager:    cloud-controller
    Operation:  Update
    Time:       2022-07-20T20:57:42Z
  Owner References:
    API Version:           crd.cloud.antrea.io/v1alpha1
    Block Owner Deletion:  true
    Controller:            true
    Kind:                  VirtualMachine
    Name:                  i-05331c205bc6df47f
    UID:                   e10dec87-6ced-40ee-8527-8a8e1869af7c
  Resource Version:        8243
  UID:                     38b5b036-10c2-4f4f-8bb7-75a25e51a1cb
Spec:
  Endpoints:
    Ip:           10.0.1.28
    Ip:           54.193.85.45
  External Node:  cloud-controller
Events:           <none>
```

## Apply Antrea NetworkPolicy

With the VMs imported into the cluster, we can now configure their security
policies by setting and applying [Antrea NetworkPolicies (ANP)](https://github.com/antrea-io/antrea/blob/main/docs/antrea-network-policy.md)
on them. The policy will be realized with cloud native security groups and
security rules. Please refer to [NetworkPolicy documentation](networkpolicy.md)
for more information on how ANPs are used, translated, and applied.

Cloud VM CRs may be selected in `externalEntitySelectors` under `To`, `From` and
`AppliedTo` fields of the Antrea `NetworkPolicy`.

The below sample ANP allows ssh traffic to all VMs.

```bash
$ cat <<EOF | kubectl apply -f -
apiVersion: crd.antrea.io/v1alpha1
kind: NetworkPolicy
metadata:
  name: vm-anp
  namespace: sample-ns
spec:
  priority: 1
  appliedTo:
  - externalEntitySelector:
      matchLabels:
         kind.nephe: virtualmachine
  ingress:
  - action: Allow
    from:
      - ipBlock:
          cidr: 0.0.0.0/0
    ports:
      - protocol: TCP
        port: 22

EOF
```

Below shows the security groups on the AWS EC2 console after the above network
policy is applied.

<img src="./assets/cloud-sg.png" width="1500" alt="CloudConsoleSGs"/>

The VirtualMachinePolicy API will display the policy realization status of all
policies being applied to a VM. The ANP status on a virtual machine will be
shown in the `Realization` field. In the below example, `vm-anp` is successfully
applied to all VMs.

```bash
$ kubectl get virtualmachinepolicy -A
$ kubectl get vmp -A
NAMESPACE   VM NAME               REALIZATION   COUNT
sample-ns   i-01b09fee2f216c1d7   SUCCESS       1
sample-ns   i-02a0b61c39cb34e5c   SUCCESS       1
sample-ns   i-0ae693c487e22dca8   SUCCESS       1
```

The `externalEntitySelector` field in ANP supports the following pre-defined
labels:

* `kind.nephe`: Select based on CRD type. Currently, only supported
  CRD types is `virtualmachine` in lower case. `virtualmachine` may be used in
  `To`, `From`, `AppliedTo` ANP fields. Thus, an ANP may be applied to virtual
  machines.
* `vpc.nephe`: Select based on cloud resources VPC.
* `name.nephe`: Select based on K8s resource name. The resource name
  is meaningful only within the K8s cluster. For AWS, virtual machine name is
  the AWS VM instance ID. For Azure virtual machine name is the hashed values of
  the Azure VM resource ID.
* `KEY.tag.nephe`: Select based on cloud resource tag key/value pair,
  where KEY is the cloud resource tag key in lower case and label value is cloud
  resource tag value in lower case.
