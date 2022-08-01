## Summary
This is a working frankenstein project written by someone that's by no means an expert in Go. It tries to solve the situation of being unaware of the hardware tier when having Kubernetes of Openshift running on VMWare.

## Tiers
The application is only aware of the pod/ container tier and the virtual machine/ openshift node tier in terms of (anti) affinity. While wanting to be able to prevent failure of application cluster and having anti affinity rules for the openshift node layer it could be that all or a (too) big of the openshift nodes in terms of VMs for VMWare are all located on the same ESX Node/ physical server. If that server fails it could mean unexpected failure of the application cluster.

     ┌───────────────────────────────────────────────────┐
     │                                                   │
     │ APPLICATION                                       │
     │                                                   │
     └───────────────────────────────────────────────────┘

     ┌───────────────────────────────────────────────────┐
     │                                                   │
     │ PODS CONTAINING CONTAINERS                        │
     │                                                   │
     └───────────────────────────────────────────────────┘

     ┌───────────────────────────────────────────────────┐
     │                                                   │
     │ VIRTUAL MACHINES CONTAINING K8S / OPENSHIFT NODES │
     │                                                   │
     └───────────────────────────────────────────────────┘

     ┌───────────────────────────────────────────────────┐
     │                                                   │
     │  PHYSCICAL SERVERS CONTAINING ESX NODES           │
     │                                                   │
     └───────────────────────────────────────────────────┘

## The solution
The solution is to label the Openshift nodes with the ESX Node they are located at. The application can then monitor that label of the Openshift nodes. In the begin scenario the application can deploy their instances spread on different Openshift ESX Nodes. If at the VMWare level the VM's/ Openshift nodes are migrated to another ESX Node for maintanance or in case of failure this program will relabel the ESX Nodes. The application can then re-spread/ re-balance their instances again.

## Packages
This project contains code examples based mainly on 2 packages.

### govmomi
This the most populair Go package for talking to the VSphere/ ESX API. In the package 'get_vms' a code example is used and modified from someone that used govmomi and was correspondig at this Github issue: https://github.com/vmware/govmomi/issues/1495

### k8s.io/client-go/kubernetes
This is the official Go package talking to an Kubernetes cluster. In the 'main' package a code example is used and modified from some that used client-go and was corresponding at this Stackoverflow issue (although hardly recognizable): https://stackoverflow.com/questions/64191539/how-to-list-all-pods-in-k8s-cluster-using-client-go-in-golang-program

### How it fits together
The main package at some point is invoking 	vms := get_vms.Show(). This function is found in the 'get_vms' package and retrieves a map return_vms[vm.Summary.Config.Name] = hname . This map contains as key the VM name, which is equal to the Openshift/ Kubernetes node, and as value the hostname. In the main package the the VM name of the corrent Openshift node is looping over is then looked up in the map as key and the value/ hostname, at Value: vms[n.Name], is then returned and used for a patch struct. This is then applied to the node as label.
```
		payload := []patchStringValue{{
			Op:    "replace",
			Path:  "/metadata/labels/esx-node",
			Value: vms[n.Name],
		}}
```
## Test setup

### K9s cluster
I have a k9s kubernetes cluster running deployed on hetzner with https://github.com/vitobotta/hetzner-k3s
It contains 2 nodes, 1 master and 1 worker node:
```
NAME                                 STATUS   ROLES                       AGE    VERSION
management-cpx11-master1             Ready    control-plane,etcd,master   190d   v1.24.1+k3s1
management-cx41-pool-small-worker1   Ready    <none>                      190d   v1.24.1+k3s1
```
### VCenter API simulation
reference: https://hub.docker.com/r/satak/vcsim
```
$ podman run -d --name vcsim -p 443:443 satak/vcsim
```
### govc binary
This is a binary that makes it easy to interact with VMWare. In the setup it is used to create 2 ESX hosts and 2 VM's that represent the nodes in my k9s cluster/ with equal names. It will make them after the vcsim container is running that is based on the latest API of VCenter.

#### create 2 ESX hosts
```
$ govc host.add -hostname esxos01 -username user -password pass
$ govc host.add -hostname esxos02 -username user -password pass
```
#### create 2 VM's
```
$ govc vm.create -m 2048 -c 2 -host=esxos01 management-cpx11-master1
$ govc vm.create -m 2048 -c 2 -host=esxos02 management-cx41-pool-small-worker1
```
## Environment variables

### Interaction with VMWare
The following environment variables are set and automatically picked up buy the packages and the govc binary to communicate with VMware and/ or 'vcsim'
```
export GOVC_URL=https://user:pass@127.0.0.1:443
export GOVC_INSECURE=true
export GOVC_DATASTORE=datastore/LocalDS_0
export GOVC_RESOURCE_POOL=DC0_H0/Resources
export GOVC_NETWORK="network/VM Network"

export GOVMOMI_INSECURE=true
export GOVMOMI_URL=https://user:pass@localhost/sdk
```
### Interaction with Kubernetes
The client-go package automatically picks up the kubeconfig that is stored at ~/.kube/config. It is also possible to let it look for a secret so that when it is running as a container on the cluster it has proper access.

## Testing the script
### 1. Check if the esx nodes and vm's are present by running govc find -l
```
$ govc find -l
Folder                       /
Datacenter                   /DC0
Folder                       /DC0/vm
...
VirtualMachine               /DC0/vm/management-cpx11-master1
VirtualMachine               /DC0/vm/management-cx41-pool-small-worker1
Folder                       /DC0/host
...
ComputeResource              /DC0/host/esxos01
HostSystem                   /DC0/host/esxos01/esxos01
ResourcePool                 /DC0/host/esxos01/Resources
ComputeResource              /DC0/host/esxos02
HostSystem                   /DC0/host/esxos02/esxos02
ResourcePool                 /DC0/host/esxos02/Resources
Folder                       /DC0/datastore
Datastore                    /DC0/datastore/LocalDS_0
Folder                       /DC0/network
Network                      /DC0/network/VM Network
DistributedVirtualSwitch     /DC0/network/DVS0
DistributedVirtualPortgroup  /DC0/network/DVS0-DVUplinks-9
DistributedVirtualPortgroup  /DC0/network/DC0_DVPG0
```

### 2. See on what ESX node one of the vm's is running (look at Host:)
```
$ govc vm.info management-cx41-pool-small-worker1
Name:           management-cx41-pool-small-worker1
  Path:         /DC0/vm/management-cx41-pool-small-worker1
  UUID:         195560ab-4e87-58f3-92ea-7a32e5491f4f
  Guest name:   otherGuest
  Memory:       2048MB
  CPU:          2 vCPU(s)
  Power state:  poweredOff
  Boot time:    <nil>
  IP address:
  Host:         esxos01
```

### 3. Run ./label_ocp_nodes
```
$ ./label_ocp_nodes
2022/08/01 21:44:06 Node management-cx41-pool-small-worker1 labelled successfully with esx-node esxos01.
```

### 4. Migrate it to another ESX host
```
$ govc vm.migrate -host /DC0/host/esxos02/esxos02 /DC0/vm/management-cx41-pool-small-worker1
```

### 5. Run ./label_ocp_nodes again
```
$ ./label_ocp_nodes
2022/08/01 21:44:06 Node management-cx41-pool-small-worker1 labelled successfully with esx-node esxos02.
```

### 6. Check the labels
```
$ oc get nodes --show-labels
NAME                                 STATUS   ROLES                       AGE    VERSION        LABELS
management-cpx11-master1             Ready    control-plane,etcd,master   190d   v1.24.1+k3s1   esx-node=esxos01
management-cx41-pool-small-worker1   Ready    <none>                      190d   v1.24.1+k3s1   esx-node=esxos02
```

### 7. Run ./label_ocp_nodes without any changes
Noting happens because in the main package it has been checked if the current situation is changed and if the VMWare API responded properly/ is not down:
```
if labels["esx-node"] != vms[n.Name] && vms[n.Name] != "" {
```