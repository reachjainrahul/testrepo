# VM Agent Deployment

This document describes VM agent deployment requirement and a reference deployment implementation.

## Requirements

### ServiceAccount
VM agent communicates with Cloud Controller and Antrea Controller in K8s cluster. For this reason
, it requires a K8s ServiceAccount that authenticate and authorize the agent. Each VM agent must
have one and only one ServiceAccount; and VM agent ServiceAccount may be shared among one or more VM
agents. 
 
The VM agent ServiceAccount binds to two roles:
1. a read-only role that allows VM agent to access Antrea Network Policies published by Antrea
 Controller.
1. a read/write role that allows VM agent to monitor and update its own VirtualMachine and
 VirtualMachineRuntimeInfo CRD.
An example of this ServiceAccount is in config/samples/demo/vmagent-setup.yaml.

### Selecting VM in Agented or Agentless Mode
An VM may run in either agented or agentless mode. Note in both cases, we assume that VM agent is
already running, and the agent can access into K8s cluster.

CloudEntitySelector selects VMs importing to K8s cluster, as well as to choose if the imported VMs
should run agented or agentless mode.
``` golang
type VirtualMachineMatch struct {
	// VpcMatch specifies the virtual private cloud to which VirtualMachines belong.
	// If it is not specified, VirtualMachines may belong to any virtual private cloud,
	VpcMatch *EntityMatch `json:"vpcMatch,omitempty"`
	// VMMatch specifies VirtualMachines to match.
	// If it is not specified, all VirtualMachines are matching.
	VMMatch *EntityMatch `json:"vmMatch,omitempty"`
	// Agented specifies if VM runs in agented mode, default is false.
	Agented bool `json:"agented,omitempty"`
	// ServiceAccount used by VM agent.
	ServiceAccount string `json:"serviceAccount,omitempty"`
}
``` 
Two new fields are added:
1. Agented is true if this VM runs agented mode.
1. ServiceAccount is the ServiceAccount name used by the VM agent.

An example of this CloudEntitySelector is in config/samples/demo/cloudaccount.yaml

### Cloud Controller Parameters
Two Cloud Controller Parameters are added.

```
    # VMAgentDefaultServiceAccount is default service account in the VM's namespace
    # that VM agent uses to authenticate itself.
    # It already exists and it must already associates with Role and RoleBinding.
    # Default is empty
    # vmAgentDefaultServiceAccount: ""

    # VMAgentAPIServerAddr is "IP:port" that VM agent uses to connection to API server.
    # Default is empty
    # vmAgentAPIServerAddr: ""

```
1. vmAgentDefaultServiceAccount, if specified, it is the default ServiceAccount that VM agent
uses if CloudEntitySelector.Spec.VirtualMachineMatch does not provide one.
1. vmAgentAPIServer is the host and port of K8s API server from VM agents' perspective. Cloud
Controller uses it to configure cloud security group rule(s) to allow VM agents communicating with
the K8s APIServer.

### Agent Requirements
VM agent requires the following:
1. openvswitch already running.
1. kubeconfig file generated from the VM agent ServiceAccount.
1. VM Agent configuration parameters
```
# VMID is VirtualMachine resource name/ID.
vmID:

# VMNamespace is VirtualMachine resource namespace.
vmNamespace:

# HostOS is operating systems OS.
hostOS:

#EgressAllowRules are default outbound rules.
egressAllowRules:

# DataPath is openvswitch datapath type.
# Default is system
dataPath:

# ExistOnDisconnect is true if agent exists upon disconnect from  APIServer.
exitOnDisconnect:
```
For instance,  VM agent may starts as 
```
./vmagent --kubeconfig=/etc/antreaplus/k8s-vmagent.conf --config=/etc/antreaplus/vmagent.conf
```

## Deployment
VM agent deployment, upgrade, support bundle collection are use case specific, and tangential to
core VM agent functionality. Nonetheless we have developed a reference deployment implementation
using combination of terraform, docker-compose to deploy VM agents on AWS ec2 instances or Azure VMs.
The remaining of this section describes that work flow.

* Install cloud tools
```` 
./hack/install-cloud-tools.sh
````    

* Deploy an eks or eks cluster, please refer to [cloud](cloud.md) for details.
* Take note of external APIService address of eks, and change the `vmAgentAPIServerAddr` field
of antrea-plus.yaml manifest. Apply the modified manifest by
```
eks kubectl apply -f config/antrea-plus.yaml
or
aks kubectl apply -f config/antrea-plus.yaml
```
* Create a VM agent ServiceAccount and associated roles.
```
eks kubectl apply -f config/samples/demo/vmagent-setup.yaml
or
aks kubectl apply -f config/samples/demo/vmagent-setup.yaml
``` 
* Create a corresponding kubeconfig from above ServiceAccount. Taking note where kubeconfig file
is stored. In this example, it is in ~/tmp/terraform-eks. 
```
KUBECONFIG=~/tmp/terraform-eks/kubeconfig ./hack/svc_to_kubeconfig.sh agent-demo vmns ~/tmp
/terraform-eks/
or
KUBECONFIG=~/tmp/terraform-aks/kubeconfig ./hack/svc_to_kubeconfig.sh agent-demo vmns ~/tmp
/terraform-aks/
```
* Push VM agent to dockerhub, assuming you have created a private repo called
docker.io/YOUR_DOCKER_ID/vmagent,
```
docker tag antrea/vmagent YOUR_DOCKER_ID/vmagent
docker push YOUR_DOCKER_ID/vmagent
```
* Instantiate AWS VPC with ec2 instances as decribed in [cloud](cloud.md). In addtion, you will
 need to have the following environment variables set.
```
export TF_VAR_aws_docker_usr=YOUR_DOCKER_ID
export TF_VAR_aws_docker_pwd=YOUR_DOCKER_PWD
export TF_VAR_aws_vm_k8s_conf=PATH_TO_SERVICE_ACCOUNT_KUBECONFIG
```
Then start deploy ec2 instance and install VM agents
```
 TF_VAR_aws_vm_with_agent=true ~/terraform/aws-tf create
```
Alternatively on aks,
```
export TF_VAR_aks_docker_usr=YOUR_DOCKER_ID
export TF_VAR_aks_docker_pwd=YOUR_DOCKER_PWD
export TF_VAR_aks_vm_k8s_conf=PATH_TO_SERVICE_ACCOUNT_KUBECONFIG
```
Then start deploy azure VM instances and install VM agents
```
 TF_VAR_azure_vm_with_agent=true ~/terraform/azure-tf create
```

* After the ec2 instances and VM agents installed (it may take a few
 minutes), 
 Import VMs by modifying VPC and agent mode in config/samples/demo/cloudaccount.yaml. Depending
 on agent mode you choose, VM should be running in agented mode
```
NAMESPACE   NAME                  CLOUD-PROVIDER   VIRTUAL-PRIVATE-CLOUD   STATUS    AGENT
vmns        i-0789ff573cf8b877f   AWS              vpc-0472385dfb74c8877   running   Running
vmns        i-0f412a065301bd6e3   AWS              vpc-0472385dfb74c8877   running   Running
vmns        i-0f83e6203a7985639   AWS              vpc-0472385dfb74c8877   running   Running
```  
or agentless mode
```
NAMESPACE   NAME                  CLOUD-PROVIDER   VIRTUAL-PRIVATE-CLOUD   STATUS    AGENT
vmns        i-0789ff573cf8b877f   AWS              vpc-0472385dfb74c8877   running   
vmns        i-0f412a065301bd6e3   AWS              vpc-0472385dfb74c8877   running   
vmns        i-0f83e6203a7985639   AWS              vpc-0472385dfb74c8877   running   
```
