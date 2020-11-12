# Air-gapped openEBS and cStor operator installation and migration 

This document will help you install openEBS with the CSI cStor operators in an airgapped installation. The examples assume that you are installing openEBS 2.2.0. If this is not the case, check the appropriate version's helm chart values.yaml and the cstor-operator yaml for the correct image versions to download.  


## Installation and migration process

1. Download helm chart from here: https://github.com/openebs/charts/releases  
2. Transfer the helm chart into your system, and make it available to your helm repo.  
3. Download the cstor-operator-2.2.0.yaml from [here](https://github.com/openebs/charts/blob/gh-pages/2.2.0/cstor-operator-2.2.0.yaml) and transfer it your system:  
4. Download the following list of images from docker hub and transfer it to your system:  
    openebs/m-apiserver:2.2.0  
    openebs/openebs-k8s-provisioner:2.2.0  
    openebs/provisioner-localpv:2.2.0  
    openebs/snapshot-controller:2.2.0  
    openebs/snapshot-provisioner:2.2.0  
    openebs/node-disk-manager:0.9.0  
    openebs/node-disk-operator:0.9.0  
    openebs/admission-server:2.2.0  
    openebs/admission-server:2.2.0  
    openebs/jiva:2.2.0  
    openebs/cstor-pool:2.2.0  
    openebs/cstor-pool-mgmt:2.2.0  
    openebs/cstor-istgt:2.2.0  
    openebs/cstor-volume-mgmt:2.2.0  
    openebs/linux-utils:2.2.0  
    openebs/m-exporter:2.2.0  
    k8scsi/csi-resizer:v0.4.0  
    k8scsi/csi-snapshotter:v2.0.1  
    k8scsi/snapshot-controller:v2.0.1  
    k8scsi/csi-provisioner:v1.6.0  
    k8scsi/csi-attacher:v2.0.0  
    k8scsi/csi-cluster-driver-registrar:v1.0.1  
    openebs/cstor-csi-driver:2.2.0  
    k8scsi/csi-node-driver-registrar:v1.0.1  
    openebs/cstor-csi-driver:2.2.0  
    openebs/cspc-operator-amd64:2.2.0  
    cstor-pool-manager-amd64:2.2.0  
    openebs/cvc-operator-amd64:2.2.0  
    openebs/cstor-volume-manager-amd64:2.2.0  
    openebs/cstor-webhook-amd64:2.2.0  
5. If you plan on migrating csp pools to cspc pools also download (and update the migration yaml to point your repo) the following image:  
    openebs/migrate-amd64:2.2.0  
6. If you use velero for backup and restore, also download this image:
    openebs/velero-plugin:2.2.0  
   Add it to velero with the following command: 
   `velero plugin add <internal_repo>/openebs/velero-plugin:2.2.0` 
   If you have a previous version of the plugin you may need to issue `velero plugin delete` to remove it.
7. Update the values.yaml to point to your repository. 
8. Edit the cstor-operator yaml. Find and replace all the `docker.io` and `quay.io` references to your internal repo. Note that some of these references will be found within environment variables.  
9. Install openEBS with `helm install` or `helm upgrade`. Then run `kubectl apply -f` on the cstor-operator manifest. 

