
<img src="/docs/workload/elasticsearch/o-elastic.png" alt="OpenEBS and Elasticsearch" style="width:400px;">

<br>

## Introduction

<br>

EFK is the most popular cloud native logging solution on Kubernetes for On-Premise as well as cloud platforms. In the EFK stack, Elasticsearch is a stateful application that needs persistent storage. Logs of production applications need to be stored for a long time which requires reliable and highly available storage.  OpenEBS and EFK together provides a complete logging solution.



Advantages of using OpenEBS for Elasticsearch database:

- All the logs data is stored locally and managed natively to Kubernetes
- Start with small storage and add disks as needed on the fly
- Logs are highly available. When a node fails or rebooted during upgrades, the persistent volumes from OpenEBS continue to be highly available. 
- If required, take backup of the Elasticsearch database periodically and back them up to S3 or any object storage so that restoration of the same logs is possible to the same or any other Kubernetes cluster

<br>

*Note: Elasticsearch can be deployed both as `deployment` or as `statefulset`. When Elasticsearch deployed as `statefulset`, you don't need to replicate the data again at OpenEBS level. When Elasticsearch is deployed as `deployment`, consider 3 OpenEBS replicas, choose the StorageClass accordingly.*

<br>

<hr>

<br>



## Deployment model

<br>



<img src="/docs/workload/elasticsearch/elasticsearch-deployment.svg" alt="OpenEBS and Elasticsearch" style="width:100%;">

<br>

<hr>

<br>

## Configuration workflow
1. [Install OpenEBS](/docs/workload/elasticsearch/elasticsearch.md#install-openebs)
2. [Select OpenEBS storage engine](/docs/workload/elasticsearch/elasticsearch.md#select-openebs-storage-engine)
3. [Configure OpenEBS cStor StorageClass](/docs/workload/elasticsearch/elasticsearch.md#configure-openebs-cstor-storageclass)
4. [Installing KUDO Operator](/docs/workload/elasticsearch/elasticsearch.md#installing-kudo-operator)
5. [Installing and Accessing Elasticsearch](/docs/workload/elasticsearch/elasticsearch.md#installing-and-accessing-elasticsearch)
6. [Installing Kibana](/docs/workload/elasticsearch/elasticsearch.md#installing-kibana)
7. [Installing Fluentd-ES](/docs/workload/elasticsearch/elasticsearch.md#installing-fluentd-es)



<br>


### Install OpenEBS

   If OpenEBS is not installed in your K8s cluster, this can done from [here](https://docs.openebs.io/docs/next/installation.html). If OpenEBS is already installed, go to the next step. 

### Select OpenEBS storage engine

A storage engine is the data plane component of the IO path of a Persistent Volume. In CAS architecture, users can choose different data planes for different application workloads based on a configuration policy. OpenEBS provides different types of storage engines and you should choose the right engine that suits your type of application requirements and storage available on your Kubernetes nodes. More information can be read from [here](https://docs.openebs.io/docs/next/overview.html#types-of-openebs-storage-engines).

After OpenEBS installation, choose the OpenEBS storage engine as per your requirement. 
- Choose **cStor**, If you are looking for replicated storage feature and other enterprise graded features such as volume expansion, backup and restore, etc. cStor configuration can be found [here](https://github.com/openebs/cstor-operators/blob/HEAD/docs/quick.md). In this document, we will be doing Elasticsearch installation using OpenEBS cStor.
- Choose **OpenEBS Local PV**, if you are not requiring replicated storage but high performance storage engine. The steps for deploying Prometheus operator using OpenEBS Local PV can be found [here](https://docs.openebs.io/docs/next/elasticsearch.html).


### Configure OpenEBS cStor StorageClass

The Storage Class `cstorsc-gqzxe`, which was created during cStor configuration, has been chosen to deploy Elasticsearch in the Kubernetes cluster. Please use the cStor storage class name which you had created during cStor configuration.


###  Installing KUDO Operator

In this section, we will install the KUDO operator. We will later deploy the latest available version of Elasticsearch using KUDO. 

Use the latest stable version of KUDO CLI. The latest version of KUDO can be checked from [here](https://github.com/kudobuilder/kudo/releases).

#### Verify if Cert-manager is installed

For installing KUDO operator, the Cert-manager must be already installed in your cluster. If not, install the Cert-manager. The instruction can be found from [here](https://cert-manager.io/docs/installation/kubernetes/#installing-with-regular-manifests). Since our K8s version is v1.18.12, we have installed Cert-manager using the following command.

```bash
kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v1.2.0/cert-manager.yaml
```

```bash
kubectl get pods --namespace cert-manager
```
```bash
#sample output
NAME                                      READY   STATUS    RESTARTS   AGE
cert-manager-7747db9d88-7qjnm             1/1     Running   0          81m
cert-manager-cainjector-87c85c6ff-whnzr   1/1     Running   0          81m
cert-manager-webhook-64dc9fff44-qww8s     1/1     Running   0          81m
```

#### Installing KUDO operator into cluster

Once prerequisites are installed you need to initialize the KUDO operator. The following command will install KUDO v0.18.2.

```bash
kubectl-kudo init --version 0.18.2
```
```bash
#sample output
$KUDO_HOME has been configured at /home/k8s/.kudo
✅ installed crds
✅ installed namespace
✅ installed service account
✅ installed webhook
✅ installed kudo controller
```

Verify pods in the `kudo-system` namespace:
```bash
kubectl get pod -n kudo-system
```
```bash
#sample output
NAME                        READY   STATUS    RESTARTS   AGE
kudo-controller-manager-0   1/1     Running   0          25s
```

#### Setting cStor Storage Class as default

Applications installed using Kudo uses the default storage class and hence change the default storage class from your current setting to OpenEBS cStor. For example, in this tutorial, the default storage class is changed to `cstorsc-gqzxe` from the default storage class `standard` in GKE. If you do not want to change the default storage class, then  pass `-p STORAGE_CLASS=cstorsc-gqzxe` to `kubectl-kudo install elastic ...` command.

```bash
kubectl patch storageclass standard -p '{"metadata": {"annotations":{"storageclass.kubernetes.io/is-default-class":"false"}}}'
kubectl patch storageclass cstorsc-gqzxe -p '{"metadata": {"annotations":{"storageclass.kubernetes.io/is-default-class":"true"}}}'
```

#### Verify default Storage Class
 
List the storage classes and verify `cstorsc-gqzxe` is set to `default`.
```bash
kubectl get sc
```
```bash
#sample output
NAME                        PROVISIONER                                                RECLAIMPOLICY   VOLUMEBINDINGMODE      ALLOWVOLUMEEXPANSION   AGE
cstorsc-gqzxe (default)     cstor.csi.openebs.io                                       Delete          Immediate              true                   139m
openebs-device              openebs.io/local                                           Delete          WaitForFirstConsumer   false                  148m
openebs-hostpath            openebs.io/local                                           Delete          WaitForFirstConsumer   false                  148m
openebs-jiva-default        openebs.io/provisioner-iscsi                               Delete          Immediate              false                  148m
openebs-snapshot-promoter   volumesnapshot.external-storage.k8s.io/snapshot-promoter   Delete          Immediate              false                  148m
premium-rwo                 pd.csi.storage.gke.io                                      Delete          WaitForFirstConsumer   true                   158m
standard                    kubernetes.io/gce-pd                                       Delete          Immediate              true                   158m
standard-rwo                pd.csi.storage.gke.io                                      Delete          WaitForFirstConsumer   true                   158m
```

### Installing and Accessing Elasticsearch

Set instance and namespace variables:
```bash
export instance_name=elastic
export namespace_name=default
kubectl-kudo install elastic --namespace=$namespace_name --instance $instance_name
```
```bash
#sample output
operator default/elastic created
operatorversion default/elastic-7.0.0-0.2.1 created
instance default/elastic created
```
#### Verifying Elastic pods

```bash
kubectl get pods -n $namespace_name
```
```bash
#sample output
NAME                    READY   STATUS    RESTARTS   AGE
elastic-coordinator-0   1/1     Running   0          31s
elastic-data-0          1/1     Running   0          56s
elastic-data-1          1/1     Running   0          44s
elastic-master-0        1/1     Running   0          2m31s
elastic-master-1        1/1     Running   0          119s
elastic-master-2        1/1     Running   0          90s
```

#### Verifying Services

```bash
kubectl get svc -n $namespace_name
```
```bash
#sample output
NAME                     TYPE        CLUSTER-IP   EXTERNAL-IP   PORT(S)    AGE
elastic-coordinator-hs   ClusterIP   None         <none>        9200/TCP   62s
elastic-data-hs          ClusterIP   None         <none>        9200/TCP   87s
elastic-ingest-hs        ClusterIP   None         <none>        9200/TCP   50s
elastic-master-hs        ClusterIP   None         <none>        9200/TCP   3m2s
kubernetes               ClusterIP   10.4.0.1     <none>        443/TCP    5h18m
```

#### Verifying PVC
```bash
kubectl get pv,pvc -n $namespace_name
```
```bash
#sample output
NAME                                                        CAPACITY   ACCESS MODES   RECLAIM POLICY   STATUS   CLAIM                                STORAGECLASS    REASON   AGE
persistentvolume/pvc-3d7cd2ff-2bd7-429a-a312-f492471939a7   2Gi        RWO            Delete           Bound    default/data-elastic-master-0        cstorsc-gqzxe            8m29s
persistentvolume/pvc-b5c37e90-fc1f-4a29-843c-6823b0f069c7   2Gi        RWO            Delete           Bound    default/data-elastic-master-1        cstorsc-gqzxe            7m16s
persistentvolume/pvc-b8e12373-7066-4e14-84de-62a5fd7585ab   2Gi        RWO            Delete           Bound    default/data-elastic-coordinator-0   cstorsc-gqzxe            3m40s
persistentvolume/pvc-c2d81474-4d3b-40e8-adde-e73861e0659e   2Gi        RWO            Delete           Bound    default/data-elastic-master-2        cstorsc-gqzxe            6m16s
persistentvolume/pvc-db44b9c2-abbf-4fc2-b7ae-9ec173cae3d6   4Gi        RWO            Delete           Bound    default/data-elastic-data-0          cstorsc-gqzxe            5m14s
persistentvolume/pvc-fdf44935-f299-4d8b-a781-f84e4546bf8b   4Gi        RWO            Delete           Bound    default/data-elastic-data-1          cstorsc-gqzxe            4m29s

NAME                                               STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS    AGE
persistentvolumeclaim/data-elastic-coordinator-0   Bound    pvc-b8e12373-7066-4e14-84de-62a5fd7585ab   2Gi        RWO            cstorsc-gqzxe   3m41s
persistentvolumeclaim/data-elastic-data-0          Bound    pvc-db44b9c2-abbf-4fc2-b7ae-9ec173cae3d6   4Gi        RWO            cstorsc-gqzxe   5m14s
persistentvolumeclaim/data-elastic-data-1          Bound    pvc-fdf44935-f299-4d8b-a781-f84e4546bf8b   4Gi        RWO            cstorsc-gqzxe   4m30s
persistentvolumeclaim/data-elastic-master-0        Bound    pvc-3d7cd2ff-2bd7-429a-a312-f492471939a7   2Gi        RWO            cstorsc-gqzxe   8m31s
persistentvolumeclaim/data-elastic-master-1        Bound    pvc-b5c37e90-fc1f-4a29-843c-6823b0f069c7   2Gi        RWO            cstorsc-gqzxe   7m16s
persistentvolumeclaim/data-elastic-master-2        Bound    pvc-c2d81474-4d3b-40e8-adde-e73861e0659e   2Gi        RWO            cstorsc-gqzxe   6m16s
```


#### Verifying Elastic instance status

```bash
kubectl kudo plan status --namespace=$namespace_name \
 --instance $instance_name
```
```bash
#sample output
Plan(s) for "elastic" in namespace "default":
.
└── elastic (Operator-Version: "elastic-7.0.0-0.2.1" Active-Plan: "deploy")
    └── Plan deploy (serial strategy) [COMPLETE], last updated 2021-06-03 20:02:21
        ├── Phase deploy-master (parallel strategy) [COMPLETE]
        │   └── Step deploy-master [COMPLETE]
        ├── Phase deploy-data (parallel strategy) [COMPLETE]
        │   └── Step deploy-data [COMPLETE]
        ├── Phase deploy-coordinator (parallel strategy) [COMPLETE]
        │   └── Step deploy-coordinator [COMPLETE]
        └── Phase deploy-ingest (parallel strategy) [COMPLETE]
            └── Step deploy-ingest [COMPLETE] 
```

#### Verify OpenEBS related components are created for Elasticsearch instances.

```bash
kubectl get pod -n openebs
```
```bash
#sample output
NAME                                                              READY   STATUS    RESTARTS   AGE
cspc-operator-f84dc4b48-rhzdd                                     1/1     Running   0          168m
cstorpool-ivwyx-j68d-6755957fb7-kcfgh                             3/3     Running   0          161m
cstorpool-ivwyx-kdbb-d9d48685f-wr6z2                              3/3     Running   0          161m
cstorpool-ivwyx-qjc4-8454b6b558-72jbn                             3/3     Running   0          161m
cvc-operator-7b8b668794-njw5d                                     1/1     Running   0          168m
maya-apiserver-7bb865b7bc-cb6v7                                   1/1     Running   0          168m
openebs-admission-server-5687dd4764-47x5l                         1/1     Running   0          168m
openebs-cstor-admission-server-bd4c5894-nrf55                     1/1     Running   0          168m
openebs-cstor-csi-controller-0                                    6/6     Running   0          168m
openebs-cstor-csi-node-28fvq                                      2/2     Running   0          168m
openebs-cstor-csi-node-dpr67                                      2/2     Running   0          168m
openebs-cstor-csi-node-kmclk                                      2/2     Running   0          168m
openebs-localpv-provisioner-5b698959d8-q6tdb                      1/1     Running   0          168m
openebs-ndm-7d98n                                                 1/1     Running   0          168m
openebs-ndm-fncbz                                                 1/1     Running   0          168m
openebs-ndm-j7rk6                                                 1/1     Running   0          168m
openebs-ndm-operator-854795b6fd-bxdtr                             1/1     Running   0          168m
openebs-provisioner-5d94599c68-k7f76                              1/1     Running   0          168m
openebs-snapshot-operator-7b6bcdbb69-x75f4                        2/2     Running   0          168m
pvc-3d7cd2ff-2bd7-429a-a312-f492471939a7-target-86cdd8689dxz2xg   3/3     Running   0          17m
pvc-b5c37e90-fc1f-4a29-843c-6823b0f069c7-target-58ff989d9bzbwbt   3/3     Running   1          16m
pvc-b8e12373-7066-4e14-84de-62a5fd7585ab-target-d44c4cb69-58srm   3/3     Running   1          12m
pvc-c2d81474-4d3b-40e8-adde-e73861e0659e-target-74b56dbfc42kgp4   3/3     Running   1          15m
pvc-db44b9c2-abbf-4fc2-b7ae-9ec173cae3d6-target-666b8fd5d7r5t97   3/3     Running   0          14m
pvc-fdf44935-f299-4d8b-a781-f84e4546bf8b-target-65cb44595-zq5qt   3/3     Running   1          13m
```

Pod names containing `target` are the cStor target pods for Elasticsearch instances.

Verify if cStor Volume Claim(CVC) are created successfully.

```bash
kubectl get cvc -n openebs
```
```bash
#sample output
NAME                                       CAPACITY   STATUS   AGE
pvc-3d7cd2ff-2bd7-429a-a312-f492471939a7   2Gi        Bound    17m
pvc-b5c37e90-fc1f-4a29-843c-6823b0f069c7   2Gi        Bound    16m
pvc-b8e12373-7066-4e14-84de-62a5fd7585ab   2Gi        Bound    13m
pvc-c2d81474-4d3b-40e8-adde-e73861e0659e   2Gi        Bound    15m
pvc-db44b9c2-abbf-4fc2-b7ae-9ec173cae3d6   4Gi        Bound    14m
pvc-fdf44935-f299-4d8b-a781-f84e4546bf8b   4Gi        Bound    13m
```

Verify if cStor Volume Replicas(CVR) for the pods are created successfully. In the Storage Class specification, number of replica is 3. So, for each CVC, Three cStor volume replicas will be created.

```bash
kubectl get cvr -n openebs
```
```bash
#sample output
NAME                                                            ALLOCATED   USED    STATUS    AGE
pvc-3d7cd2ff-2bd7-429a-a312-f492471939a7-cstorpool-ivwyx-j68d   1.55M       13.4M   Healthy   18m
pvc-3d7cd2ff-2bd7-429a-a312-f492471939a7-cstorpool-ivwyx-kdbb   1.55M       13.4M   Healthy   18m
pvc-3d7cd2ff-2bd7-429a-a312-f492471939a7-cstorpool-ivwyx-qjc4   1.55M       13.4M   Healthy   18m
pvc-b5c37e90-fc1f-4a29-843c-6823b0f069c7-cstorpool-ivwyx-j68d   2.51M       19.1M   Healthy   16m
pvc-b5c37e90-fc1f-4a29-843c-6823b0f069c7-cstorpool-ivwyx-kdbb   2.51M       19.1M   Healthy   16m
pvc-b5c37e90-fc1f-4a29-843c-6823b0f069c7-cstorpool-ivwyx-qjc4   2.51M       19.1M   Healthy   16m
pvc-b8e12373-7066-4e14-84de-62a5fd7585ab-cstorpool-ivwyx-j68d   89.5K       2.84M   Healthy   13m
pvc-b8e12373-7066-4e14-84de-62a5fd7585ab-cstorpool-ivwyx-kdbb   89.5K       2.84M   Healthy   13m
pvc-b8e12373-7066-4e14-84de-62a5fd7585ab-cstorpool-ivwyx-qjc4   90K         2.85M   Healthy   13m
pvc-c2d81474-4d3b-40e8-adde-e73861e0659e-cstorpool-ivwyx-j68d   2.58M       19.3M   Healthy   15m
pvc-c2d81474-4d3b-40e8-adde-e73861e0659e-cstorpool-ivwyx-kdbb   2.58M       19.3M   Healthy   15m
pvc-c2d81474-4d3b-40e8-adde-e73861e0659e-cstorpool-ivwyx-qjc4   2.58M       19.3M   Healthy   15m
pvc-db44b9c2-abbf-4fc2-b7ae-9ec173cae3d6-cstorpool-ivwyx-j68d   7.12M       27.9M   Healthy   14m
pvc-db44b9c2-abbf-4fc2-b7ae-9ec173cae3d6-cstorpool-ivwyx-kdbb   7.12M       27.9M   Healthy   14m
pvc-db44b9c2-abbf-4fc2-b7ae-9ec173cae3d6-cstorpool-ivwyx-qjc4   7.12M       27.9M   Healthy   14m
pvc-fdf44935-f299-4d8b-a781-f84e4546bf8b-cstorpool-ivwyx-j68d   7.01M       27.1M   Healthy   14m
pvc-fdf44935-f299-4d8b-a781-f84e4546bf8b-cstorpool-ivwyx-kdbb   7.01M       27.1M   Healthy   14m
pvc-fdf44935-f299-4d8b-a781-f84e4546bf8b-cstorpool-ivwyx-qjc4   7.01M       27.1M   Healthy   14m
```

#### Accessing Elasticsearch

Enter into one of the master pod using exec command:
```bash
kubectl exec -it elastic-master-0 -- bash
```
```bash
#sample output
[root@elastic-master-0 elasticsearch]#
```
Run below command inside Elastic master pod:
```bash
curl -X POST "elastic-coordinator-hs:9200/twitter/_doc/" -H 'Content-Type: application/json' -d'
 {
     "user" : "openebs",
     "post_date" : "2021-03-02T14:12:12",
     "message" : "Test data entry"
 }'
```
Following is the output of the above command:
```bash
#sample output
{"_index":"twitter","_type":"_doc","_id":"LoliyXcBg9iVzVnOj5QL","_version":1,"result":"created","_shards":{"total":2,"successful":1,"failed":0},"_seq_no":0,"_primary_term":1}
[root@elastic-master-0 elasticsearch]#
```

The above command added data into Elasticsearch. You can use the following command to query for the inserted data:
```bash
curl -X GET "elastic-coordinator-hs:9200/twitter/_search?q=user:openebs&pretty"
```
```bash
#sample output
{
  "took" : 141,
  "timed_out" : false,
  "_shards" : {
    "total" : 1,
    "successful" : 1,
    "skipped" : 0,
    "failed" : 0
  },
  "hits" : {
    "total" : {
      "value" : 1,
      "relation" : "eq"
    },
    "max_score" : 0.2876821,
    "hits" : [
      {
        "_index" : "twitter",
        "_type" : "_doc",
        "_id" : "qoWznXkBpvRSYN1fECvZ",
        "_score" : 0.2876821,
        "_source" : {
          "user" : "openebs",
          "post_date" : "2021-03-02T14:12:12",
          "message" : "Test data entry"
        }
      }
    ]
  }
}
```

Now, let's get the details of Elasticsearch cluster. The cluster information will show the Elasticsearch version, cluster name and other details. If you are getting similar information, then it means your Elasticsearch deployment is successful.

```bash
curl localhost:9200
```
```bash
#sample output
{
  "name" : "elastic-master-0",
  "cluster_name" : "elastic-cluster",
  "cluster_uuid" : "A0qErYmCS2OpJtmgR_j3ow",
  "version" : {
    "number" : "7.10.1",
    "build_flavor" : "default",
    "build_type" : "docker",
    "build_hash" : "1c34507e66d7db1211f66f3513706fdf548736aa",
    "build_date" : "2020-12-05T01:00:33.671820Z",
    "build_snapshot" : false,
    "lucene_version" : "8.7.0",
    "minimum_wire_compatibility_version" : "6.8.0",
    "minimum_index_compatibility_version" : "6.0.0-beta1"
  },
  "tagline" : "You Know, for Search"
}
```


### Installing Kibana

First, add helm repository of Elastic.
```bash
helm repo add elastic https://helm.elastic.co && helm repo update
```

Install Kibana deployment using `helm` command. Ensure to meet required prerequisites corresponding to your helm version. Fetch the Kibana `values.yaml`:
```bash
wget https://raw.githubusercontent.com/elastic/helm-charts/master/kibana/values.yaml
``` 

Edit the following parameters:

1. `elasticsearchHosts` as `"http://elastic-coordinator-hs:9200"` # service name of Elastic search.
2. `service.type` as `"NodePort"`.
3. `service.nodePort` as `"30295"` # since this port is already added in our network firewall rules.
4. `imageTag` as `"7.10.1"` , it should be the same image tag of Elasticsearch. In our case,  Elasticsearch image tag is 7.10.1.


Now install Kibana using Helm:
```bash
helm install  kibana -f values.yaml elastic/kibana
```

Verifying Kibana Pods and Services:
```bash
kubectl get pod
```
```bash
#sample output
NAME                             READY   STATUS    RESTARTS   AGE
elastic-coordinator-0            1/1     Running   0          9m13s
elastic-data-0                   1/1     Running   0          8m27s
elastic-data-1                   1/1     Running   0          8m36s
elastic-master-0                 1/1     Running   0          9m54s
elastic-master-1                 1/1     Running   0          10m
elastic-master-2                 1/1     Running   0          11m
kibana-kibana-757865c4d8-7lzlg   1/1     Running   0          2m56s
```

```bash
kubectl get svc
```
```bash
#sample output
NAME                     TYPE        CLUSTER-IP     EXTERNAL-IP   PORT(S)          AGE
elastic-coordinator-hs   ClusterIP   None           <none>        9200/TCP         18m
elastic-data-hs          ClusterIP   None           <none>        9200/TCP         19m
elastic-ingest-hs        ClusterIP   None           <none>        9200/TCP         18m
elastic-master-hs        ClusterIP   None           <none>        9200/TCP         20m
kibana-kibana            NodePort    10.4.6.94      <none>        5601:30295/TCP   7m1s
kubernetes               ClusterIP   10.4.0.1       <none>        443/TCP          5h36m
```



### Installing Fluentd-ES

Fetch the `values.yaml`:
```bash
wget https://raw.githubusercontent.com/bitnami/charts/master/bitnami/fluentd/values.yaml
helm repo add bitnami https://charts.bitnami.com/bitnami && helm repo update
```

Replace the following section in the `values.yaml` file with new content.

Old:
```yaml   
     # Send the logs to the standard output
      <match **>
        @type stdout
      </match>
```
New:
```yaml
      # Send the logs to the elasticsearch output
      <match **>
        @type elasticsearch
        include_tag_key true
        host elastic-coordinator-hs
        port 9200
        logstash_format true

        <buffer>
          @type file
          path /opt/bitnami/fluentd/logs/buffers/logs.buffer
          flush_thread_count 2
          flush_interval 5s
        </buffer>
      </match>
 ```
 
 
 
Install Fluentd-Elasticsearch DaemonSet using the new values:
```bash
helm install fluentd -f values.yaml bitnami/fluentd
```

Verify Fluentd Daemonset, Pods and Services:
```bash
kubectl get ds
```
```bash
#sample output
NAME      DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR   AGE
fluentd   3         3         3       3            3           <none>          74m
```
```bash
kubectl get pod
```
```bash
#sample output
NAME                                 READY   STATUS    RESTARTS   AGE
pod/elastic-coordinator-0            1/1     Running   0          16m
pod/elastic-data-0                   1/1     Running   0          15m
pod/elastic-data-1                   1/1     Running   0          15m
pod/elastic-master-0                 1/1     Running   0          16m
pod/elastic-master-1                 1/1     Running   0          17m
pod/elastic-master-2                 1/1     Running   0          18m
pod/fluentd-0                        1/1     Running   0          118s
pod/fluentd-4dgdd                    1/1     Running   2          118s
pod/fluentd-j495w                    1/1     Running   2          118s
pod/fluentd-sp5l4                    1/1     Running   2          118s
pod/kibana-kibana-757865c4d8-7lzlg   1/1     Running   0          9m52s
```

```bash
kubectl get svc
```
```bash
#sample output
NAME                             TYPE        CLUSTER-IP   EXTERNAL-IP   PORT(S)              AGE
service/elastic-coordinator-hs   ClusterIP   None         <none>        9200/TCP             17m
service/elastic-data-hs          ClusterIP   None         <none>        9200/TCP             19m
service/elastic-ingest-hs        ClusterIP   None         <none>        9200/TCP             16m
service/elastic-master-hs        ClusterIP   None         <none>        9200/TCP             22m
service/fluentd-aggregator       ClusterIP   10.4.7.78    <none>        9880/TCP,24224/TCP   2m1s
service/fluentd-forwarder        ClusterIP   10.4.0.2     <none>        9880/TCP             2m1s
service/fluentd-headless         ClusterIP   None         <none>        9880/TCP,24224/TCP   2m1s
service/kibana-kibana            NodePort    10.4.6.94    <none>        5601:30295/TCP       9m55s
service/kubernetes               ClusterIP   10.4.0.1     <none>        443/TCP              3h2m
```



Getting logs from the indices:

1. Goto Kibana dashboard.
2. Click on Management->Stack Management which is placed at left side bottom most.
3. Click on index patterns listed under Kibana and then click on `Create Index pattern`.
4. Provide `logstash-*` inside the index pattern box and then select `Next step`.
5. In the next step, inside the `Time Filter` field name, select the `@timestamp` field from the dropdown menu, and click `Create index pattern`.
6. Now click on the `Discover` button listed on the top left of the side menu bar.
7. There will be a dropdown menu where you can select the available indices.
8. In this case, you have to select `logstash-*` from the dropdown menu.


Now let's do some tests:

If you want to get the logs of NDM pods, type the following text inside the `Filters` field.
`kubernetes.labels.openebs_io/component-name.keyword : "ndm"` and then choose the required date and time period. After that, click Apply.
You will see the OpenEBS NDM pod logs listed on the page.

<br>


## See Also:

### [cStor User guide](https://github.com/openebs/cstor-operators/blob/HEAD/docs/quick.md)
### [Troubleshooting cStor](https://github.com/openebs/cstor-operators/blob/HEAD/docs/troubleshooting/troubleshooting.md)

<br>

<hr>
