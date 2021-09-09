<img src="/docs/workload/prometheus-operator/o-prometheus.png" alt="OpenEBS and Prometheus" style="width:400px;">

## Introduction


Each and every DevOps and SRE's are looking for ease of deployment of their applications in Kubernetes. After successful installation, they will be looking for how easily it can be monitored to maintain the availability of applications in a real-time manner. They can take proactive measures before an issue arise by monitoring the application. Prometheus is the mostly widely used application for scraping cloud native application metrics. Prometheus and OpenEBS together provide a complete open source stack for monitoring.

In this document, we will explain how you can easily set up a monitoring environment in your K8s cluster using Prometheus and use OpenEBS CSI based cStor PV as the persistent storage for storing the metrics. Since Prometheus is non-distributed application, using cStor helps in having data availability in case of node failures, due to the cStor's replicated storage nature. This guide provides the installation of Prometheus using Helm on dynamically provisioned OpenEBS cStor volumes. 

## Deployment model



<img src="/docs/workload/prometheus-operator/prometheus-deployment.svg" alt="OpenEBS and Prometheus using cStor" style="width:100%;">



We will add a 100G disk to each node. These disks will be consumed by CSI based cStor pool and later Prometheus and Alert manager instances will be using these pools to create their persistent volumes. The recommended configuration is to have at least three nodes for provisioning at least 3 volume replicas for each volume and an unclaimed external disk to be attached per node to create cStor storage pool on each node with striped manner .  



## Configuration workflow

1. [Meet Prerequisites](/docs/workload/prometheus-operator/prometheus.md#meet-prerequisites)

4. [Installing Prometheus Operator](/docs/workload/prometheus-operator/prometheus.md#installing-prometheus-operator)

5. [Accessing Prometheus and Grafana](/docs/workload/prometheus-operator/prometheus.md#accessing-prometheus-and-grafana)

   

### Meet Prerequisites

OpenEBS should be installed first on your Kubernetes cluster. The steps for OpenEBS installation can be found [here](https://docs.openebs.io/docs/next/installation.html). After OpenEBS installation, choose the OpenEBS storage engine as per your requirement. 

- Choose **cStor**, If you are looking for replicated storage feature and other enterprise graded features such as volume expansion, backup and restore, etc. cStor configuration can be found [here](https://github.com/openebs/cstor-operators/blob/HEAD/docs/quick.md). In this document, we are mentioning about the Prometheus operator installation using OpenEBS cStor.
- Choose **OpenEBS Local PV**, if you are not requiring replicated storage but high performance storage engine. The steps for deploying Prometheus operator using OpenEBS Local PV can be found [here](https://docs.openebs.io/docs/next/prometheus.html).

### Installing Prometheus Operator

In this section, we will install the Prometheus operator using OpenEBS cStor storage engine. The following are the high-level overview to install Prometheus operator on cStor.

- Fetch and update Prometheus repository
- Configure Prometheus Helm `values.yaml`
- Create namespace for installing application
- Install Prometheus operator

#### Fetch and update Prometheus Helm repository

```
$ helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
$ helm repo update
```

#### Configure Prometheus Helm `values.yaml`

Download `values.yaml` which we will modify before installing Prometheus using Helm.

```none
wget https://raw.githubusercontent.com/prometheus-community/helm-charts/main/charts/kube-prometheus-stack/values.yaml
```

Perform the following changes:

- Update `fullnameOverride: "app" `

- Update `prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues` as `false`

- Update `prometheus.prometheusSpec.replicas` as `1`

- Uncomment the following spec in the `values.yaml` for `prometheus.prometheusSpec.storageSpec` and change the StorageClass name of Prometheus with the required StorageClass and required volume capacity. In this case, Storage Class used as `openebs-device` , provided the volume capacity as `40Gi`  and added `metdata.name` as `prom`. 

   ```
   storageSpec:
     volumeClaimTemplate:
       metadata:
         name: prom
       spec:
         storageClassName: cstor-csi
         accessModes: ["ReadWriteOnce"]
         resources:
           requests:
             storage: 40Gi
   ```

- (Optional in case of GKE) Update `prometheusOperator.admissionWebhooks.enabled` as `false`. More details can be found [here](https://github.com/mayadata-io/website-mayadata/pull/994).

- Update `prometheusOperator.tls.enabled` as `false`

- Update `alertmanager.alertmanagerSpec.replicas` as `1`

- Uncomment the following spec in the `values.yaml` for `alertmanager.alertmanagerSpec.storage` and change the StorageClass name of Alert manager with the required StorageClass name and required volume capacity. In this case, StorageClass used as `openebs-device` and provided the volume capacity as `40Gi` and added `metdata.name` as `alert`. 

  ```none
   storage:
      volumeClaimTemplate:
      metadata:
        name: alert
      spec:
        storageClassName: cstor-csi
        accessModes: ["ReadWriteOnce"]
        resources:
          requests:
            storage: 40Gi
  ```

#### Create namespace for installing Prometheus operator

```
$ kubectl create ns monitoring
```

#### Install Prometheus operator

The following command will install both Prometheus and Grafana components.
```
$ helm install prometheus prometheus-community/kube-prometheus-stack --namespace monitoring -f values.yaml
```
**Note:** Check compatibility for your Kubernetes version and Prometheus stack from [here](https://github.com/prometheus-operator/kube-prometheus/tree/release-0.7#compatibility).



Verify Prometheus related pods are installed under **monitoring** namespace

```
$ kubectl get pod -n monitoring

NAME                                             READY   STATUS    RESTARTS   AGE
alertmanager-app-alertmanager-0                  2/2     Running   0          5m30s
app-operator-7cf8fc6dc-k6wb6                     1/1     Running   0          5m40s
prometheus-app-prometheus-0                      2/2     Running   1          5m30s
prometheus-grafana-6549f869b5-7dvp4              2/2     Running   0          5m40s
prometheus-kube-state-metrics-685b975bb7-f9qt6   1/1     Running   0          5m40s
prometheus-prometheus-node-exporter-6ngps        1/1     Running   0          5m40s
prometheus-prometheus-node-exporter-fnfbt        1/1     Running   0          5m40s
prometheus-prometheus-node-exporter-mlvt6        1/1     Running   0          5m40s
```

Verify Prometheus related PVCs are created under **monitoring** namespace

```
$ kubectl get pvc -n monitoring

NAME                                    STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS   AGE
alert-alertmanager-app-alertmanager-0   Bound    pvc-d734b059-e80a-488f-b398-c66e0b3c208c   40Gi       RWO            cstor-csi      5m53s
prom-prometheus-app-prometheus-0        Bound    pvc-24bbb044-c080-4f66-a0f6-e51cded91286   40Gi       RWO            cstor-csi      5m53s
```
Verify OpenEBS related components are created for Prometheus and Alert manager instances.

```
root@ranjith-vm:~/prometheus# kubectl get pod -n openebs
NAME                                                              READY   STATUS    RESTARTS   AGE
cspc-operator-f5dcc784d-t5w2s                                     1/1     Running   0          35m
cstor-storage-6ccv-5574cb8f9f-f6mg2                               3/3     Running   0          32m
cstor-storage-gpm8-7577fb9df9-647bj                               3/3     Running   0          32m
cstor-storage-v8t9-69bbcf75cb-9t596                               3/3     Running   0          32m
cvc-operator-7cf7ffb7c8-jpcjx                                     1/1     Running   0          35m
maya-apiserver-58bf9f895-jj4mg                                    1/1     Running   0          94m
openebs-admission-server-757767dd44-st9zl                         1/1     Running   0          94m
openebs-cstor-admission-server-7b955dfccd-kw4kp                   1/1     Running   0          35m
openebs-cstor-csi-controller-0                                    6/6     Running   0          35m
openebs-cstor-csi-node-2tlzl                                      2/2     Running   0          35m
openebs-cstor-csi-node-lwxk4                                      2/2     Running   0          35m
openebs-cstor-csi-node-xzw2b                                      2/2     Running   0          35m
openebs-localpv-provisioner-867c9f8bd4-z789v                      1/1     Running   0          94m
openebs-ndm-7gqgt                                                 1/1     Running   0          35m
openebs-ndm-operator-8674d458dd-ml258                             1/1     Running   0          35m
openebs-ndm-pm9l6                                                 1/1     Running   0          34m
openebs-ndm-wqg7c                                                 1/1     Running   0          35m
openebs-provisioner-6566699f56-82m2n                              1/1     Running   0          94m
openebs-snapshot-operator-6f5fbd98d6-5pvrv                        2/2     Running   0          94m
pvc-24bbb044-c080-4f66-a0f6-e51cded91286-target-7b564fcfd-lmt4v   3/3     Running   0          17m
pvc-d734b059-e80a-488f-b398-c66e0b3c208c-target-66c6454854tv6rj   3/3     Running   1          17m
```

Pod names containing `target` are the cStor target pods for Prometheus and Alert manager instances.

Verify if cStor Volume Claim(CVC) are created successfully.

```
kubectl get cvc -n openebs
NAME                                       CAPACITY   STATUS   AGE
pvc-24bbb044-c080-4f66-a0f6-e51cded91286   40Gi       Bound    18m
pvc-d734b059-e80a-488f-b398-c66e0b3c208c   40Gi       Bound    18m
```

Verify if cStor Volume Replicas(CVR) for both pods are created successfully. In the Storage Class specification, number of replica is 3. So, for each CVC, Three cStor volume replicas will be created.

```
kubectl get cvr -n openebs
NAME                                                          ALLOCATED   USED    STATUS    AGE
pvc-24bbb044-c080-4f66-a0f6-e51cded91286-cstor-storage-6ccv   8.62M       29.4M   Healthy   18m
pvc-24bbb044-c080-4f66-a0f6-e51cded91286-cstor-storage-gpm8   8.62M       29.4M   Healthy   18m
pvc-24bbb044-c080-4f66-a0f6-e51cded91286-cstor-storage-v8t9   8.62M       29.4M   Healthy   18m
pvc-d734b059-e80a-488f-b398-c66e0b3c208c-cstor-storage-6ccv   232K        16.8M   Healthy   18m
pvc-d734b059-e80a-488f-b398-c66e0b3c208c-cstor-storage-gpm8   232K        16.8M   Healthy   18m
pvc-d734b059-e80a-488f-b398-c66e0b3c208c-cstor-storage-v8t9   232K        16.8M   Healthy   18m
```

Verify Prometheus related services created under **monitoring** namespace

```
$ kubectl get svc -n monitoring

NAME                                  TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)                      AGE
alertmanager-operated                 ClusterIP   None            <none>        9093/TCP,9094/TCP,9094/UDP   15m
app-alertmanager                      ClusterIP   10.100.21.20    <none>        9093/TCP                     16m
app-operator                          ClusterIP   10.100.95.182   <none>        8080/TCP                     16m
app-prometheus                        ClusterIP   10.100.67.27    <none>        9090/TCP                     16m
prometheus-grafana                    ClusterIP   10.100.39.64    <none>        80/TCP                       16m
prometheus-kube-state-metrics         ClusterIP   10.100.72.188   <none>        8080/TCP                     16m
prometheus-operated                   ClusterIP   None            <none>        9090/TCP                     15m
prometheus-prometheus-node-exporter   ClusterIP   10.100.250.3    <none>        9100/TCP                     16m
```



For ease of simplicity in testing the deployment, we are going to use NodePort for **prometheus-kube-prometheus-prometheus** and **prometheus-grafana** services . Please be advised to consider using **LoadBalancer** or **Ingress**, instead of **NodePort**, for production deployment.



Change Prometheus service to **NodePort** from **ClusterIP**:

```
$ kubectl patch svc app-prometheus -n monitoring -p '{"spec": {"type": "NodePort"}}'
```



Change prometheus-grafana service to **LoadBalancer**/**NodePort** from **ClusterIP**:

```
$ kubectl patch svc prometheus-grafana -n monitoring -p '{"spec": {"type": "NodePort"}}'
```

**Note:** If the user needs to access Prometheus and Grafana outside the network, the service type can be changed or a new service should be added to use Load Balancer or create Ingress resources for production deployment.

Sample output after making the above 2 changes in services:

```
$ kubectl get svc -n monitoring

NAME                                  TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)                      AGE
alertmanager-operated                 ClusterIP   None            <none>        9093/TCP,9094/TCP,9094/UDP   20m
app-alertmanager                      ClusterIP   10.100.21.20    <none>        9093/TCP                     20m
app-operator                          ClusterIP   10.100.95.182   <none>        8080/TCP                     20m
app-prometheus                        NodePort    10.100.67.27    <none>        9090:30203/TCP               20m
prometheus-grafana                    NodePort    10.100.39.64    <none>        80:30489/TCP                 20m
prometheus-kube-state-metrics         ClusterIP   10.100.72.188   <none>        8080/TCP                     20m
prometheus-operated                   ClusterIP   None            <none>        9090/TCP                     20m
prometheus-prometheus-node-exporter   ClusterIP   10.100.250.3    <none>        9100/TCP                     20m
```



### Accessing Prometheus and Grafana

Get the node details using the following command:

```
$ kubectl get node -o wide

NAME                                            STATUS   ROLES    AGE    VERSION   INTERNAL-IP      EXTERNAL-IP      OS-IMAGE             KERNEL-VERSION   CONTAINER-RUNTIME
ip-192-168-22-4.ap-south-1.compute.internal     Ready    <none>   100m   v1.19.5   192.168.22.4     15.206.124.193   Ubuntu 20.04.2 LTS   5.4.0-1037-aws   docker://19.3.8
ip-192-168-59-204.ap-south-1.compute.internal   Ready    <none>   100m   v1.19.5   192.168.59.204   13.127.98.221    Ubuntu 20.04.2 LTS   5.4.0-1037-aws   docker://19.3.8
ip-192-168-77-39.ap-south-1.compute.internal    Ready    <none>   100m   v1.19.5   192.168.77.39    52.66.223.8      Ubuntu 20.04.2 LTS   5.4.0-1037-aws   docker://19.3.8
```




Verify Prometheus service is accessible over web browser using http://<any_node_external-ip:<NodePort>

Example:

http://15.206.124.193:30203/

**Note**: It may be required to allow the Node Port number/traffic in the Firewall/Security Groups to access the above Grafana and Prometheus URL on the web browser.




Launch Grafana using Node's External IP and with corresponding NodePort of **prometheus-grafana** service

http://<any_node_external-ip>:<Grafana_SVC_NodePort>

Example:

http://15.206.124.193:30489/



Grafana Credentials:

**Username**: admin

**Password**: prom-operator



Password can be obtained using the command:

```
$ (kubectl get secret \
    --namespace monitoring prometheus-grafana \
    -o jsonpath="{.data.admin-password}" \
    | base64 --decode ; echo
)
```

The above credentials need to be provided when you need to access the Grafana console. Login to your Grafana console using the above credentials.

Users can upload a Grafana dashboard for Prometheus in 3 ways. 

- First method is by proving the Grafana id of the corresponding dashboard and then load it.
  Just find the Grafana dashboard id of the Prometheus and then just mention this id and then load it. The Grafana dashboard id of Prometheus is **3681**.
  
- Another approach is download the following Grafana dashboard JSON file for Prometheus and then paste it in the console and then load it. 

  ```
  https://raw.githubusercontent.com/FUSAKLA/Prometheus2-grafana-dashboard/master/dashboard/prometheus2-dashboard.json

- The other way to monitor Prometheus Operator is by using the inbuilt Prometheus dashboard. This can be obtained by searching on the Grafana dashboard and finding the Prometheus dashboard under the General category.


<br>

## See Also:

### [cStor User guide](https://github.com/openebs/cstor-operators/blob/HEAD/docs/quick.md)

### [Troubleshooting cStor](https://github.com/openebs/cstor-operators/blob/HEAD/docs/troubleshooting/troubleshooting.md)

<br>

<hr>
