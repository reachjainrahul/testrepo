# Users Guide

## Installation on single cluster

Prerequisites:
* Assume there exists a cluster reachable via kubectl.
* Install Antrea on KinD cluster,
```bash
kubectl apply -f ci/kind/antrea-kind-anp.yaml
```
* Install Antrea on aws eks,
```bash
kubectl apply -f ci/eks/antrea-eks-anp.yaml
```
* Install Antrea on azure aks,
```bash
kubectl apply -f ci/aks/antrea-aks-anp.yaml
```

Install AntreaPlus,
```bash
kubectl apply -f config/antrea-plus.yaml
```

## Import Cloud Resources

### CloudAccount
Users need to first configure a corresponding CloudAccount CRD from which cloud resources are
 imported.
For instance, this applies an aws CloudAccount,  
```bash
cat <<EOF | kubectl apply -f -
apiVersion: cloud.antreaplus.tanzu.vmware.com/v1alpha1
kind: CloudProviderAccount
metadata:
  name: cloudprovideraccount-sample
spec:
  providerType: "AWS"
  configAWS:
    accessKeyId: "MY_AWS_KEY"
    accessKeySecret: "MY_AWS_SECRET"
    region: "us-west-2"
EOF
``` 

And this applies an azure CloudAccount,
```bash
cat <<EOF | kubectl apply -f -
apiVersion: cloud.antreaplus.tanzu.vmware.com/v1alpha1
kind: CloudProviderAccount
metadata:
  name: cloudprovideraccount-sample
  namespace: vmns
spec:
  providerType: "Azure"
  configAzure:
    subscriptionId: "MY_AZURE_SUBS_ID"
    clientId: "MY_AZURE_CLIENT_ID"
    tenantId: "MY_AZURE_TENANT_ID"
    clientKey: "MY_AZURE_CLIENT_KEY"
    region:    "westus2"
EOF
``` 

TBD. Explaining how to use each fields 

### CloudEntitySelector
Once a CloudAccount is created, cloud resources, such as VirtualMachines and
NativeServices, may be imported CloudAccount's Namespace via CloudEntitySelector 

This example selects VirtualMachines in VPC `MY_VPC_ID` from CloudAccount 
`cloudprovideraccount-sample` to import.
```bash
cat <<EOF | kubectl apply -f -
apiVersion: cloud.antreaplus.tanzu.vmware.com/v1alpha1
kind: CloudEntitySelector
metadata:
  name: cloudentityselector-sample01
spec:
  accountName: cloudprovideraccount-sample
  vmSelector:
    vmMatches:
      - vpcMatch:
          matchID: "MY_VPC_ID"
EOF
``` 
If there are virtual machines running in `MY_VPC_ID`, these virtual machines and associated NICs
 should be imported. i.e.
```bash
kubectl get virtualmachines -A
NAMESPACE   NAME                  CLOUD-PROVIDER   VIRTUAL-PRIVATE-CLOUD   STATUS    AGENT
vmns        i-01b09fee2f216c1d7   AWS              vpc-02d3e1e0f15a56f4b   running   
vmns        i-02a0b61c39cb34e5c   AWS              vpc-02d3e1e0f15a56f4b   running   
vmns        i-0ae693c487e22dca8   AWS              vpc-02d3e1e0f15a56f4b   running   
kubectl get networkinterfaces -A
NAMESPACE   NAME                    OWNER-ID              OWNER-TYPE       INTERNAL-IP   EXTERNAL-IP
vmns        eni-002850663e78704bd   i-01b09fee2f216c1d7   VirtualMachine   10.0.1.66     54.187.86.146
vmns        eni-0062fbc40a61db49f   i-02a0b61c39cb34e5c   VirtualMachine   10.0.1.75     52.35.36.78
vmns        eni-08ebe9c0da67f0db0   i-0ae693c487e22dca8   VirtualMachine   10.0.1.158    54.185.185.255
```

For now, the following matches for selecting VirtualMachines are supported:
* AWS:
  * vpcMatch: matchID, matchName
  * vmMatch: matchID, matchName
* Azure:
  * vpcMatch: matchID
  * vmMatch: matchID, matchName
 
TBD.  Explaining how to use each fields

### Import VM as Service

An VM may be imported as Services via tag
```bash
svc.user.antreaplus.tanzu.vmware.com = SERVICE:PORT,...
```
A VM can expose multiple Services separated by `,`, multiple VMs can expose the same Services if
 the Service name is the same. 
 
In above example, AntreaPlus generates `SERVICE-pub` and/or `SERVICE-priv` Services, where 
`SERVICE-pub` refers to the Endpoints consists of VMs' public IPs, and `SERVICE-priv` refers to
 the Endpoints consists of VMs' private IPs. Pods in this cluster can DNS
 resolve these Services, and VM can be added or remove from these Sevrices
 via changes to tagging, and VM IPs can be ephemeral. 

## Apply Antrea Network Policy

The cloud resource CRDs may be referenced ExternalEntitySelectors in `To`, `From`, `AppliedTo
` fields 
[Antrea NetworkPolicy (ANP)](https://github.com/antrea-io/antrea/blob/main/docs/antrea-network-policy.md).

For instance, the following ANP allows *curl* Pods to send traffic to all the VirtualMachines.  
```bash
cat <<EOF | kubectl apply -f -
apiVersion: crd.antrea.io/v1alpha1
kind: NetworkPolicy
metadata:
  name: pod-anp
spec:
  priority: 1
  appliedTo:
  - podSelector:
      matchLabels:
        app: curl
  egress:
  - action: Allow
    to:
    - externalEntitySelector:
        matchLabels:
           kind.antreaplus.tanzu.vmware.com: virtualmachine
  - action: Drop
    to:
      - ipBlock:
          cidr: 0.0.0.0/0
EOF
```

The following ANP allows all VirtualMachines to receive traffic from other VirtualMachines.
```bash
cat <<EOF | kubectl apply -f -
apiVersion: crd.antrea.io/v1alpha1
kind: NetworkPolicy
metadata:
  name: vm-anp
spec:
  priority: 2
  appliedTo:
  - externalEntitySelector:
      matchLabels:
         kind.antreaplus.tanzu.vmware.com: virtualmachine
  ingress:
  - action: Allow
    from:
    - externalEntitySelector:
        matchLabels:
          kind.antreaplus.tanzu.vmware.com: virtualmachine
EOF
```

Users may also check application status ANP on VirtualMachines. In the following example, an ANP
`vm-anp` is successfully applied to all VirtualMachines.
```
kubectl get virtualmachine -A -o jsonpath='{.items[*].status.networkPolicies}' 
map[vm-anp:applied] map[vm-anp:applied] map[vm-anp:applied]
```

The externalEntitySelector in ANP supports the following label keys,
* `kind.antreaplus.tanzu.vmware.com`: Select based on CRD type. The supported CRD types are, all
 in lower case, `virtualmachine`, `nativeservice`, `networkinterface`, `service
 `.  `virtualmachine` and `networkinterface` may be used in `To`, `From`, `AppliedTo` ANP fields
 ; whereas `nativeservice`, `service` may be used in `To`, 'From' ANP fields.  Thus,
 an ANP may be applied to virtual machines, network interfaces, but not services.  In addition
 , only Service of type LoadBalancer is supported. This allows ANP to control virtual machines
  traffic to K8s Services exposed via cloud provider LoadBalancer.
* `vpc.cloud.antreaplus.tanzu.vmware.com`: Select based on cloud resources VPC.      
* `name.antreaplus.tanzu.vmware.com`: Select based on K8s resource name. The resource name
 is meaningful only within the K8s cluster. The aws
 VirtualMachine and NetworkInterface name are corresponding VM instance ID and NIC interface ID
 . The azure  VirtualMachine and NetworkInrterface name are hashed values of their corresponding
  resource ID.
* `KEY.tag.antreaplus.tanzu.vmware.com`: Select based on cloud resource tag key/value pair, where
 KEY is the lower case cloud resource tag key, and label value is the lower case cloud resource
 tag value. 
 
## VM with Agent

TBD.

##Federation
We recommend to use interactive CLI to help to configure Antrea Federations. The interactive CLI
 comes with auto-completion feature.

```bash
./bin/cli
```

####Discover clusters

In the following example, two clusters kind-kind, kind-kind1 are discovered.
```bash
AntreaPlus>> discover-clusters
Retrieve kubeconfig files from /home/suw/.kube
Move old kubeconfig to /tmp/kubeconfig.2021-04-19T10:00:29-07:00.bak
I0419 10:00:31.178101 1802196 request.go:621] Throttling request took 1.041656625s, request: GET:https://172.21.0.3:6443/apis/clusterinformation.antrea.io/v1beta1?timeout=32s
Discovered cluster kind-kind in context kind-kind support federation true 
Discovered cluster kind-kind1 in context kind-kind1 support federation true 
```

###Work with clusters

Users can list all discovered clusters
```bash
AntreaPlus>> list-clusterContexts
CURRENT   NAME         CLUSTER      AUTHINFO     NAMESPACE
*         kind-kind    kind-kind    kind-kind
          kind-kind1   kind-kind1   kind-kind1
```

Users can access resources in each cluster the same way as if using kubectl with --context, 
for instance,
```bash
AntreaPlus>> cluster kind-kind get pod -n antrea-plus-system
NAME                                            READY   STATUS    RESTARTS   AGE
antrea-plus-cloud-controller-67844858f4-zg7hn   1/1     Running   1          43h
```

####Configure federation
The following commands add discovered clusters kind-kind and kind-kind1 to a federation test-fed
, and assign cluster kind-kind as supervisor cluster. For federation to function correctly, it
 must consists of at least one supervisor cluster.
```bash
AntreaPlus>> federation test-fed add kind-kind kind-kind1
AntreaPlus>> federation test-fed set cluster kind-kind --supervisor
Setup federations in supervisor clusters [kind-kind]
```

All federated sources in each cluster may viewed holistically via get/describe. For instance, the
 following commands display the federation status.
```bash
AntreaPlus>> federation test-fed describe federation
kind-kind:
federation:
Name:         test-fed
Namespace:    antrea-plus-system
Labels:       <none>
Annotations:  <none>
API Version:  federation.antreaplus.tanzu.vmware.com/v1alpha1
Kind:         Federation
Metadata:
  Creation Timestamp:  2021-04-19T17:00:00Z
  Generation:          1
  Managed Fields:
    API Version:  federation.antreaplus.tanzu.vmware.com/v1alpha1
    Fields Type:  FieldsV1
    fieldsV1:
      f:metadata:
        f:annotations:
          .:
          f:kubectl.kubernetes.io/last-applied-configuration:
      f:spec:
        .:
        f:members:
    Manager:      kubectl-client-side-apply
    Operation:    Update
    Time:         2021-04-19T17:00:00Z
    API Version:  federation.antreaplus.tanzu.vmware.com/v1alpha1
    Fields Type:  FieldsV1
    fieldsV1:
      f:status:
        .:
        f:memberStatus:
    Manager:         antrea-plus-controller
    Operation:       Update
    Time:            2021-04-19T17:00:05Z
  Resource Version:  1164534
  UID:               dc989816-f997-4db1-9a33-124842ea0352
Spec:
  Members:
    Cluster ID:  kind-kind
    Secret:      supervisor-for-kind-kind
    Server:      https://172.21.0.3:6443
    Cluster ID:  kind-kind1
    Secret:      supervisor-for-kind-kind1
    Server:      https://172.21.0.6:6443
Status:
  Member Status:
    Cluster ID:  kind-kind
    Error:       none
    Is Leader:   true
    Status:      Reachable
    Cluster ID:  kind-kind1
    Error:       none
    Is Leader:   true
    Status:      Reachable
Events:          <none>

kind-kind1:
AntreaPlus>> 
```

Users can list all known federations
```bash
AntreaPlus>> list-federations
   test-fed
   	kind-kind
   	kind-kind1
```

User can remove a cluster from federation via
```bash
AntreaPlus>> federation test-fed remove kind-kind1
```
When there is no clusters in the federation, the federation is automatically removed.

### Configure federated Service

AntreaPlus federated Service is a Service that can be discovered via DNS query by all member cluster
 in the federation, and can be by federated Antrea NetworkPolicy. It is configured as annotation
 to a standard K8s Service. For instance this federated Service configuration in config/samples
 /demo/fed-onprem-service.yaml. It indicates the Service httpbin is a member of federated Service
  fed-httpbin.
```bash
apiVersion: v1
kind: Service
metadata:
  annotations:
    svc.fed.user.antreaplus.tanzu.vmware.com: fed-httpbin
  name: httpbin
  namespace: podns
  labels:
    app: httpbin
spec:
  type: NodePort
  ports:
    - name: http
      port: 8000
      targetPort: 80
  selector:
    app: httpbin
```

The httpbin Service may be applied to one or more clusters,
```
AntreaPlus>> cluster kind-kind apply -f config/samples/demo/fed-onprem-service.yaml
```
The federated Service and associated Endpoints are automatically generated on all clusters in
federation.
```bash
ntreaPlus>> federation test-fed get service
kind-kind1:
service:
NAME          TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)          AGE
fed-httpbin   ClusterIP   10.96.135.89    <none>        8000/TCP         7m49s
httpbin       NodePort    10.96.155.198   <none>        8000:30602/TCP   7m49s

kind-kind:
service:
NAME          TYPE        CLUSTER-IP    EXTERNAL-IP   PORT(S)    AGE
fed-httpbin   ClusterIP   10.96.82.32   <none>        8000/TCP   7m49s
```

```bash
AntreaPlus>> federation test-fed get endpoints
kind-kind:
endpoints:
NAME          ENDPOINTS                                            AGE
fed-httpbin   172.21.0.6:30602,172.21.0.7:30602,172.21.0.5:30602   7m51s

kind-kind1:
endpoints:
NAME          ENDPOINTS                     AGE
fed-httpbin   10.10.1.53:80,10.10.2.73:80   7m51s
```

###Configure federated Antrea NetworkPolicy

Antreaplus federated NetworkPolicy is configured by adding annotation to a standard
 Antrea NetworkPolicy. The federated NetworkPolicy is only applied to Pods or VMs in the same
  Namespace and cluster in which it is created. But the ExternalEntity
  selector in To/From fields an Antrea NetworkPolicy. it may select federated Services or cloud
  resources from a different cluster in the federation.

For instance, the following federated NetworkPolicy allows Egress traffic to federated Service fed
-httpbin
```
apiVersion: crd.antrea.io/v1alpha1
kind: NetworkPolicy
metadata:
  annotations:
    anp.fed.user.antreaplus.tanzu.vmware.com: "true"
  name: pod-anp
  namespace: podns
spec:
  priority: 1
  appliedTo:
    - podSelector:
        matchLabels:
          app: curl
  egress:
    - action: Allow
      to:
        - podSelector: {}
        - externalEntitySelector:
            matchLabels:
              kind.antreaplus.tanzu.vmware.com: federated-service
              name.antreaplus.tanzu.vmware.com: fed-httpbin-lb
    - action: Drop
      to:
      - ipBlock:
        cidr: 0.0.0.0/0
```
