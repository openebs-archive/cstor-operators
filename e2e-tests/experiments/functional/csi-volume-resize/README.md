## Experiment Metadata

| Type       | Description                                                  | Storage | Applications | K8s Platform |
| ---------- | ------------------------------------------------------------ | ------- | ------------ | ------------ |
| Functional | Increase the storage capacity of volume created using CSI. | OpenEBS, LocalPV| Any          | Any          |

## Entry-Criteria

- K8s nodes should be ready.
- cStor CSPC should be created in case of cstor volume resize.
- Application should be deployed using CSI provisioner.
- The feature-gate `--feature-gates=ExpandCSIVolumes` should be set to true.
- The module `iscsi_tcp` should be enabled.

## Exit-Criteria

- Volume should be resized successfully and application should be accessible seamlessly.
- Application should be able to use the new space.

## Notes

- This functional test checks if the csi volume can be resized.
- This e2e-test accepts the parameters in form of job environmental variables.
- This job updates the storage capacity of PVC with the new capacity and configures the PVC to increase its size.
- It checks if the application is accessible and the corresponding file system is scaled up.

## e2e-test Environment Variables

| Parameters    | Description                                            |
| ------------- | ------------------------------------------------------ |
| APP_NAMESPACE | Namespace where application and volume is deployed.    |
| APP_PVC       | Name of PVC whose storage capacity has to be increased |
| APP_LABEL     | Label of application pod                               |
| PV_CAPACITY   | Current storage capacity of volume                     |
| NEW_CAPACITY  | Desired storage capacity of volume                     |

