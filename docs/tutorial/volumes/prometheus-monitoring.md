Monitoring with Prometheus
==========================

### Prerequisite

We need to have k8s version 1.15+ to access the CSI volume metrics.

### Setup helm

This step uses helm the kubernetes package manager. If you have not setup the helm then do the below configuration, otherwise move to the next step.

```
$ helm version
version.BuildInfo{Version:"v3.1.2", GitCommit:"d878d4d45863e42fd5cff6743294a11d28a9abce", GitTreeState:"clean", GoVersion:"go1.13.8"}
```

### Install Prometheus Operator

To install the stable/prometheus-operator, use the Prometheus chart from the helm repository. If all goes as expected, you should have the following pods running/ready in your default namespace:

```
$ helm install prometheus-operator stable/prometheus-operator
.
.
.
NAME: prometheus-operator
LAST DEPLOYED: Tue Sep 15 19:30:48 2020
NAMESPACE: default
STATUS: deployed
REVISION: 1
NOTES:
The Prometheus Operator has been installed. Check its status by running:
  kubectl --namespace default get pods -l "release=prometheus-operator"

  Visit https://github.com/coreos/prometheus-operator for instructions on how
  to create & configure Alertmanager and Prometheus instances using the Operator.
```

Check all the required pods are up and running

```
$ kubectl get pods
NAME                                                      READY   STATUS    RESTARTS   AGE
alertmanager-prometheus-operator-alertmanager-0           2/2     Running   0          92s
percona-123-56b9995b84-5bswf                              1/1     Running   0          61m
prometheus-operator-grafana-77d5b9b8b8-xrfv7              2/2     Running   0          99s
prometheus-operator-kube-state-metrics-69fcc8d48c-pk6cz   1/1     Running   0          99s
prometheus-operator-operator-6db7d95977-wl46p             2/2     Running   0          99s
prometheus-operator-prometheus-node-exporter-7j4dp        1/1     Running   0          99s
prometheus-prometheus-operator-prometheus-0               3/3     Running   1          82s
```

### Setup alert rule

Please check the rules there in the system :-

```
$ kubectl get PrometheusRule
NAME                                                       AGE
prometheus-operator-alertmanager.rules                     4m21s
prometheus-operator-etcd                                   4m21s
prometheus-operator-general.rules                          4m21s
prometheus-operator-k8s.rules                              4m21s
prometheus-operator-kube-apiserver-error                   4m21s
prometheus-operator-kube-apiserver.rules                   4m21s
prometheus-operator-kube-prometheus-node-recording.rules   4m21s
prometheus-operator-kube-scheduler.rules                   4m21s
prometheus-operator-kubernetes-absent                      4m21s
prometheus-operator-kubernetes-apps                        4m21s
prometheus-operator-kubernetes-resources                   4m21s
prometheus-operator-kubernetes-storage                     4m21s
prometheus-operator-kubernetes-system                      4m21s
prometheus-operator-kubernetes-system-apiserver            4m21s
prometheus-operator-kubernetes-system-controller-manager   4m21s
prometheus-operator-kubernetes-system-kubelet              4m21s
prometheus-operator-kubernetes-system-scheduler            4m21s
prometheus-operator-node-exporter                          4m21s
prometheus-operator-node-exporter.rules                    4m21s
prometheus-operator-node-network                           4m21s
prometheus-operator-node-time                              4m21s
prometheus-operator-node.rules                             4m21s
prometheus-operator-prometheus                             4m21s
prometheus-operator-prometheus-operator                    4m21s
```

You can edit any of the default rules or setup the new rule to get the alerts. Here is the sample rule to start firing the alerts if available storage space is less than 10% :-

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  labels:
    app: prometheus-operator
    chart: prometheus-operator-9.3.2
    release: prometheus-operator
  name: prometheus-operator-cstor-alertmanager.rules
  namespace: default
spec:
  groups:
  - name: cstoralertmanager.rules
    rules:
    - alert: CStorVolumeUsageCritical
      annotations:
        message: The PersistentVolume claimed by {{ $labels.persistentvolumeclaim
          }} in Namespace {{ $labels.namespace }} is only {{ printf "%0.2f" $value
          }}% free.
      expr: |
        100 * kubelet_volume_stats_available_bytes{job="kubelet"}
          /
        kubelet_volume_stats_capacity_bytes{job="kubelet"}
          < 10
      for: 1m
      labels:
        severity: critical
```

Apply the above yaml so that Prometheus can generate the alert when available space is less than 10%


### Check the Prometheus alert

To be able to view the Prometheus web UI, expose it through a Service. A simple way to do this is to use a Service of type NodePort

```
$ cat prometheus-service.yaml

apiVersion: v1
kind: Service
metadata:
  name: prometheus-service
spec:
  type: NodePort
  ports:
  - name: web
    nodePort: 30090
    port: 9090
    protocol: TCP
    targetPort: web
  selector:
    prometheus: prometheus-operator-prometheus
```

apply the above yaml

```
$ kubectl apply -f prometheus-service.yaml
service/prometheus-service created
```

Now you can access the alert manager UI via "nodes-external-ip:30090"

```
$ kubectl get nodes -owide
NAME                                    STATUS   ROLES    AGE    VERSION          INTERNAL-IP   EXTERNAL-IP   OS-IMAGE             KERNEL-VERSION   CONTAINER-RUNTIME
gke-cstor-default-pool-3e407350-xvzp    Ready    <none>   103m   v1.17.4-gke.22   10.168.0.45   34.94.3.140   Ubuntu 18.04.3 LTS   5.0.0-1022-gke   docker://19.3.2
```

In this case we can access the alert manager via url http://34.94.3.140:30090/

### Check the Alert Manager

To be able to view the Alert Manager web UI, expose it through a Service of type NodePort

```
$ cat alertmanager-service.yaml

apiVersion: v1
kind: Service
metadata:
  name: alertmanager-service
spec:
  type: NodePort
  ports:
  - name: web
    nodePort: 30093
    port: 9093
    protocol: TCP
    targetPort: web
  selector:
    alertmanager: prometheus-operator-alertmanager
```

apply the above yaml

```
$ kubectl apply -f alertmanager-service.yaml
service/alertmanager-service created
```

Now you can access the alert manager UI via "nodes-external-ip:30093"

```
$ kubectl get nodes -owide
NAME                                   STATUS   ROLES    AGE    VERSION          INTERNAL-IP   EXTERNAL-IP   OS-IMAGE             KERNEL-VERSION   CONTAINER-RUNTIME
gke-cstor-default-pool-3e407350-xvzp   Ready    <none>   103m   v1.17.4-gke.22   10.168.0.45   34.94.3.140   Ubuntu 18.04.3 LTS   5.0.0-1022-gke   docker://19.3.2
```

In this case we can access the alert manager via url http://34.94.3.140:30093/

### CSI metrics exposed 

We can create the other rules for cstor metrics also. Here is the list of csi volume metrics exposed by k8s:-

```
kubelet_volume_stats_available_bytes
kubelet_volume_stats_capacity_bytes
kubelet_volume_stats_inodes
kubelet_volume_stats_inodes_free
kubelet_volume_stats_inodes_used
kubelet_volume_stats_used_bytes
```
