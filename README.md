# cstor-operators
[![Go Report](https://goreportcard.com/badge/github.com/openebs/cstor-operators)](https://goreportcard.com/report/github.com/openebs/cstor-operators)
[![Build Status](https://github.com/openebs/cstor-operators/actions/workflows/build.yml/badge.svg)](https://github.com/openebs/cstor-operators/actions/workflows/build.yml)
[![Slack](https://img.shields.io/badge/JOIN-SLACK-blue)](https://kubernetes.slack.com/messages/openebs/)

<img width="300" align="right" alt="OpenEBS Logo" src="https://raw.githubusercontent.com/cncf/artwork/master/projects/openebs/stacked/color/openebs-stacked-color.png" xmlns="http://www.w3.org/1999/html">

Collection of enhanced Kubernetes Operators for managing OpenEBS cStor Data Engine.

## Project Status: Beta

The cStor operators work in conjunction with the [cStor CSI driver](https://github.com/openebs/cstor-csi) to
provide cStor volumes for stateful workloads.

The new cStor Operators support the following Operations on cStor pools and volumes:
1. Provisioning and De-provisioning of cStor pools.
2. Pool expansion by adding disk.
3. Disk replacement by removing a disk.
4. Volume replica scale up and scale down.
5. Volume resize.
6. Backup and Restore via Velero-plugin.
7. Seamless upgrades of cStor Pools and Volumes
8. Support migration from old cStor operators (using SPC) to new cStor operators using CSPC and CSI Driver. 


## Usage

- [Quickly deploy it on K8s and get started](docs/quick.md)
- [Pool and Volume Operations Tutorial](docs/tutorial/intro.md)
- [FAQ and Troubleshooting](docs/troubleshooting/troubleshooting.md)

## Operators Overview

At a high-level, cstor operators consist of following components.
- cspc-operator
- pool-manager
- cvc-operator
- volume-manager

An OpenEBS admin/user can use CSPC(CStorPoolCluster) API (YAML) to provision cStor pools in a Kubernetes cluster.
As the name suggests, CSPC can be used to create a cluster of cStor pools across Kubernetes nodes.
It is the job of **cspc-operator** to reconcile the CSPC object and provision CStorPoolInstance(s) as specified 
in the CSPC. A cStor pool is provisioned on node by utilising the disks attached to the node and is represented by 
CStorPoolInstance(CSPI) custom resource in a Kubernetes cluster. One has freedom to specify the disks that they
want to use for pool provisioning.

CSPC API comes with a variety of tunables and features and the API can be viewed for [here](https://github.com/openebs/api/blob/master/pkg/apis/cstor/v1/cstorpoolcluster.go)

Once a CSPC is created, cspc-operator provision CSPI CR and **pool-manager** deployment on each node where cStor pool should 
be created. The pool-manager deployment watches for its corresponding CSPI on the node and finally executes commands to
perform pool operations e.g pool provisioning.

More info on cStor Pool CRs can be found [here](docs/developer-guide/cstor-pool.md).

**Note:** It is not recommended to modify the CSPI CR and pool-manager in the running cluster unless you know what you are 
trying to do. CSPC should be the only point of interaction.

Once the CStor pool(s) get provisioned successfully after creating CSPC, admin/user can create PVC to provision csi CStor volumes. When a user
creates PVC, CStor CSI driver creates CStorVolumeConfig(CVC) resource, managed and reconciled by the **cvc-controller** which creates
different volume-specific resources for each persistent volume, later managed by their respective controllers, more info
can be found [here](docs/developer-guide/cstor-volume.md).


## Raising Issues And PRs

If you want to raise any issue for cstor-operators please do that at [openebs/openebs].

## Contributing

If you would like to contribute to code and are unsure about how to proceed, 
please get in touch with maintainers in Kubernetes Slack #openebs [channel]. 

Please read the contributing guidelines [here](./CONTRIBUTING.md).

## Code of conduct

Please read the community code of conduct [here](./CODE_OF_CONDUCT.md).

[Docker environment]: https://docs.docker.com/engine
[Go environment]: https://golang.org/doc/install
[openebs/openebs]: https://github.com/openebs/openebs
[channel]: https://kubernetes.slack.com/messages/openebs/
