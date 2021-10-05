# cStor Operators
[![Go Report](https://goreportcard.com/badge/github.com/openebs/cstor-operators)](https://goreportcard.com/report/github.com/openebs/cstor-operators)
[![Build Status](https://github.com/openebs/cstor-operators/actions/workflows/build.yml/badge.svg)](https://github.com/openebs/cstor-operators/actions/workflows/build.yml)
[![Slack](https://img.shields.io/badge/JOIN-SLACK-blue)](https://kubernetes.slack.com/messages/openebs/)
[![Community Meetings](https://img.shields.io/badge/Community-Meetings-blue)](https://openebs.io/community)
[![Gitpod ready-to-code](https://img.shields.io/badge/Gitpod-ready--to--code-908a85?logo=gitpod)](https://gitpod.io/#https://github.com/openebs/cstor-operators)

<img width="300" align="right" alt="OpenEBS Logo" src="https://raw.githubusercontent.com/cncf/artwork/master/projects/openebs/stacked/color/openebs-stacked-color.png" xmlns="http://www.w3.org/1999/html">

## Project Status: Beta

We are always happy to list users who run cStor in production, check out our existing [adopters](https://github.com/openebs/openebs/tree/HEAD/adopters), and their [feedbacks](https://github.com/openebs/openebs/issues/2719).

The new cStor Operators support the following Operations on cStor pools and volumes:
1. Provisioning and De-provisioning of cStor pools.
2. Pool expansion by adding disk.
3. Disk replacement by removing a disk.
4. Volume replica scale up and scale down.
5. Volume resize.
6. Backup and Restore via Velero-plugin.
7. Seamless upgrades of cStor Pools and Volumes
8. Support migration from old cStor operators (using SPC) to new cStor operators using CSPC and CSI Driver. 

## Operators Overview

Collection of enhanced Kubernetes Operators for managing OpenEBS cStor Data Engine.
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

CSPC API comes with a variety of tunables and features and the API can be viewed for [here](https://github.com/openebs/api/blob/HEAD/pkg/apis/cstor/v1/cstorpoolcluster.go)

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

The cStor operators work in conjunction with the [cStor CSI driver](https://github.com/openebs/cstor-csi) to
provide cStor volumes for stateful workloads.


### Minimum Supported Versions

K8S : 1.18+

## Usage

- [Quickly deploy it on K8s and get started](docs/quick.md)
- [Pool and Volume Operations Tutorial](docs/tutorial/intro.md)
- [FAQ and Troubleshooting](docs/troubleshooting/troubleshooting.md)


## Raising Issues And PRs

If you want to raise any issue for cstor-operators please do that at [openebs/openebs].

## Contributing

OpenEBS welcomes your feedback and contributions in any form possible.

- [Join OpenEBS community on Kubernetes Slack](https://kubernetes.slack.com)
  - Already signed up? Head to our discussions at [#openebs](https://kubernetes.slack.com/messages/openebs/)
- Want to raise an issue or help with fixes and features?
  - See [open issues](https://github.com/openebs/openebs/issues)
  - See [contributing guide](./CONTRIBUTING.md)
  - See [Project Roadmap](https://github.com/orgs/openebs/projects/9)
  - Want to join our contributor community meetings, [check this out](https://hackmd.io/mfG78r7MS86oMx8oyaV8Iw?view).
- Join our OpenEBS CNCF Mailing lists
  - For OpenEBS project updates, subscribe to [OpenEBS Announcements](https://lists.cncf.io/g/cncf-openebs-announcements)
  - For interacting with other OpenEBS users, subscribe to [OpenEBS Users](https://lists.cncf.io/g/cncf-openebs-users)

## Code of conduct

Please read the community code of conduct [here](./CODE_OF_CONDUCT.md).

[Docker environment]: https://docs.docker.com/engine
[Go environment]: https://golang.org/doc/install
[openebs/openebs]: https://github.com/openebs/openebs
