# Prerequisites to run CStor-CSI in rancher based clusters

## Intro
CStor-CSI provides block volume support through the iSCSI protocol. Therefore, the iSCSI client(initiator) presence on all Kubernetes nodes is required.

## Step to be performed on Rancher based K8s cluster

- If you are using RancherOS as the operating system, you need to enable the iSCSI service and start iSCSI service on all the worker nodes.
- If you are using Ubuntu or RHEL as the operating system, you need to
    - Verify and install if iSCSI initiators are not installed on all the nodes.
    - Add the extra_binds under Kubelet service in cluster YAML file to mount the iSCSI binary and configuration inside the kubelet.

### iSCSI services On RancherOS
To run iSCSI services, execute the following commands on each of the cluster hosts or nodes.
```sh
sudo ros s enable open-iscsi
sudo ros s up open-iscsi
```
Run the below commands on all the nodes to make sure that the below directories are persistent, by default these directories are ephemeral.
```sh
ros config set rancher.services.user-volumes.volumes  [/home:/home,/opt:/opt,/var/lib/kubelet:/var/lib/kubelet,/etc/kubernetes:/etc/kubernetes,/var/openebs]
system-docker rm all-volumes
reboot
```

### iSCSI services on RHEL or Ubuntu

#### Step1:  Verify if iSCSI initiator is installed and services are running

Below commands are required to verify and install iscsi services on nodes

| OPERATING SYSTEM | ISCSI PACKAGE         | COMMANDS                                                 |
| ---------------- | --------------------- | -------------------------------------------------------- |
| RHEL/CentOS      | iscsi-initiator-utils | <ul><li>sudo yum install iscsi-initiator-utils -y</li><li>sudo systemctl enable --now iscsid</li><li>modprobe iscsi_tcp</li><li>echo iscsi_tcp >/etc/modules-load.d/iscsi-tcp.conf</li></ul> |
| Ununtu/ Debian   | open-iscsi            |  <ul><li>sudo apt install open-iscsi</li><li>sudo systemctl enable --now iscsid</li><li>modprobe iscsi_tcp</li><li>echo iscsi_tcp >/etc/modules-load.d/iscsi-tcp.conf</li></ui>|

#### Step2: Add extra_binds under kubelet service in cluster YAML

After installing the iSCSI initiator on your nodes, bind them into the kubelet container by editing rancher cluster.yaml, as shown in the sample below.
```sh
services:
    kubelet: 
      extra_binds: 
        - "/etc/iscsi:/etc/iscsi"
        - "/sbin/iscsiadm:/sbin/iscsiadm"
        - "/var/lib/iscsi:/var/lib/iscsi"
        - "/lib/modules"
```
**Note**: After performing prerequisites follow the steps mentioned [here](../quick.md) to deploy cStor in cluster.
