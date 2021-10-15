<p>Packages:</p>
<ul>
<li>
<a href="#cstor.openebs.io%2fv1">cstor.openebs.io/v1</a>
</li>
</ul>
<h2 id="cstor.openebs.io/v1">cstor.openebs.io/v1</h2>
<p>
<p>Package v1 is the v1 version of the API.</p>
</p>
Resource Types:
<ul><li>
<a href="#cstor.openebs.io/v1.CStorPoolCluster">CStorPoolCluster</a>
</li><li>
<a href="#cstor.openebs.io/v1.CStorPoolInstance">CStorPoolInstance</a>
</li><li>
<a href="#cstor.openebs.io/v1.CStorVolume">CStorVolume</a>
</li><li>
<a href="#cstor.openebs.io/v1.CStorVolumeConfig">CStorVolumeConfig</a>
</li><li>
<a href="#cstor.openebs.io/v1.CStorVolumePolicy">CStorVolumePolicy</a>
</li><li>
<a href="#cstor.openebs.io/v1.CStorVolumeReplica">CStorVolumeReplica</a>
</li></ul>
<h3 id="cstor.openebs.io/v1.CStorPoolCluster">CStorPoolCluster
</h3>
<p>
<p>CStorPoolCluster describes a CStorPoolCluster custom resource.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
cstor.openebs.io/v1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>CStorPoolCluster</code></td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
<a href="#cstor.openebs.io/v1.CStorPoolClusterSpec">
CStorPoolClusterSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>pools</code></br>
<em>
<a href="#cstor.openebs.io/v1.PoolSpec">
[]PoolSpec
</a>
</em>
</td>
<td>
<p>Pools is the spec for pools for various nodes
where it should be created.</p>
</td>
</tr>
<tr>
<td>
<code>resources</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#resourcerequirements-v1-core">
Kubernetes core/v1.ResourceRequirements
</a>
</em>
</td>
<td>
<p>DefaultResources are the compute resources required by the cstor-pool
container.
If the resources at PoolConfig is not specified, this is written
to CSPI PoolConfig.</p>
</td>
</tr>
<tr>
<td>
<code>auxResources</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#resourcerequirements-v1-core">
Kubernetes core/v1.ResourceRequirements
</a>
</em>
</td>
<td>
<p>AuxResources are the compute resources required by the cstor-pool pod
side car containers.</p>
</td>
</tr>
<tr>
<td>
<code>tolerations</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#toleration-v1-core">
[]Kubernetes core/v1.Toleration
</a>
</em>
</td>
<td>
<p>Tolerations, if specified, are the pool pod&rsquo;s tolerations
If tolerations at PoolConfig is empty, this is written to
CSPI PoolConfig.</p>
</td>
</tr>
<tr>
<td>
<code>priorityClassName</code></br>
<em>
string
</em>
</td>
<td>
<p>DefaultPriorityClassName if specified applies to all the pool pods
in the pool spec if the priorityClass at the pool level is
not specified.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#cstor.openebs.io/v1.CStorPoolClusterStatus">
CStorPoolClusterStatus
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>versionDetails</code></br>
<em>
<a href="#cstor.openebs.io/v1.VersionDetails">
VersionDetails
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="cstor.openebs.io/v1.CStorPoolInstance">CStorPoolInstance
</h3>
<p>
<p>CStorPoolInstance describes a cstor pool instance resource.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
cstor.openebs.io/v1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>CStorPoolInstance</code></td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
<a href="#cstor.openebs.io/v1.CStorPoolInstanceSpec">
CStorPoolInstanceSpec
</a>
</em>
</td>
<td>
<p>Spec is the specification of the cstorpoolinstance resource.</p>
<br/>
<br/>
<table>
<tr>
<td>
<code>hostName</code></br>
<em>
string
</em>
</td>
<td>
<p>HostName is the name of kubernetes node where the pool
should be created.</p>
</td>
</tr>
<tr>
<td>
<code>nodeSelector</code></br>
<em>
map[string]string
</em>
</td>
<td>
<p>NodeSelector is the labels that will be used to select
a node for pool provisioning.
Required field</p>
</td>
</tr>
<tr>
<td>
<code>poolConfig</code></br>
<em>
<a href="#cstor.openebs.io/v1.PoolConfig">
PoolConfig
</a>
</em>
</td>
<td>
<p>PoolConfig is the default pool config that applies to the
pool on node.</p>
</td>
</tr>
<tr>
<td>
<code>dataRaidGroups</code></br>
<em>
<a href="#cstor.openebs.io/v1.RaidGroup">
[]RaidGroup
</a>
</em>
</td>
<td>
<p>DataRaidGroups is the raid group configuration for the given pool.</p>
</td>
</tr>
<tr>
<td>
<code>writeCacheRaidGroups</code></br>
<em>
<a href="#cstor.openebs.io/v1.RaidGroup">
[]RaidGroup
</a>
</em>
</td>
<td>
<p>WriteCacheRaidGroups is the write cache raid group.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#cstor.openebs.io/v1.CStorPoolInstanceStatus">
CStorPoolInstanceStatus
</a>
</em>
</td>
<td>
<p>Status is the possible statuses of the cstorpoolinstance resource.</p>
</td>
</tr>
<tr>
<td>
<code>versionDetails</code></br>
<em>
<a href="#cstor.openebs.io/v1.VersionDetails">
VersionDetails
</a>
</em>
</td>
<td>
<p>VersionDetails is the openebs version.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cstor.openebs.io/v1.CStorVolume">CStorVolume
</h3>
<p>
<p>CStorVolume describes a cstor volume resource created as custom resource</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
cstor.openebs.io/v1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>CStorVolume</code></td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
<a href="#cstor.openebs.io/v1.CStorVolumeSpec">
CStorVolumeSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>capacity</code></br>
<em>
k8s.io/apimachinery/pkg/api/resource.Quantity
</em>
</td>
<td>
<p>Capacity represents the desired size of the underlying volume.</p>
</td>
</tr>
<tr>
<td>
<code>targetIP</code></br>
<em>
string
</em>
</td>
<td>
<p>TargetIP IP of the iSCSI target service</p>
</td>
</tr>
<tr>
<td>
<code>targetPort</code></br>
<em>
string
</em>
</td>
<td>
<p>iSCSI Target Port typically TCP ports 3260</p>
</td>
</tr>
<tr>
<td>
<code>iqn</code></br>
<em>
string
</em>
</td>
<td>
<p>Target iSCSI Qualified Name.combination of nodeBase</p>
</td>
</tr>
<tr>
<td>
<code>targetPortal</code></br>
<em>
string
</em>
</td>
<td>
<p>iSCSI Target Portal. The Portal is combination of IP:port (typically TCP ports 3260)</p>
</td>
</tr>
<tr>
<td>
<code>replicationFactor</code></br>
<em>
int
</em>
</td>
<td>
<p>ReplicationFactor represents number of volume replica created during volume
provisioning connect to the target</p>
</td>
</tr>
<tr>
<td>
<code>consistencyFactor</code></br>
<em>
int
</em>
</td>
<td>
<p>ConsistencyFactor is minimum number of volume replicas i.e. <code>RF/2 + 1</code>
has to be connected to the target for write operations. Basically more then
50% of replica has to be connected to target.</p>
</td>
</tr>
<tr>
<td>
<code>desiredReplicationFactor</code></br>
<em>
int
</em>
</td>
<td>
<p>DesiredReplicationFactor represents maximum number of replicas
that are allowed to connect to the target. Required for scale operations</p>
</td>
</tr>
<tr>
<td>
<code>replicaDetails</code></br>
<em>
<a href="#cstor.openebs.io/v1.CStorVolumeReplicaDetails">
CStorVolumeReplicaDetails
</a>
</em>
</td>
<td>
<p>ReplicaDetails refers to the trusty replica information</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#cstor.openebs.io/v1.CStorVolumeStatus">
CStorVolumeStatus
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>versionDetails</code></br>
<em>
<a href="#cstor.openebs.io/v1.VersionDetails">
VersionDetails
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="cstor.openebs.io/v1.CStorVolumeConfig">CStorVolumeConfig
</h3>
<p>
<p>CStorVolumeConfig describes a cstor volume config resource created as
custom resource. CStorVolumeConfig is a request for creating cstor volume
related resources like deployment, svc etc.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
cstor.openebs.io/v1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>CStorVolumeConfig</code></td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
<a href="#cstor.openebs.io/v1.CStorVolumeConfigSpec">
CStorVolumeConfigSpec
</a>
</em>
</td>
<td>
<p>Spec defines a specification of a cstor volume config required
to provisione cstor volume resources</p>
<br/>
<br/>
<table>
<tr>
<td>
<code>capacity</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#resourcelist-v1-core">
Kubernetes core/v1.ResourceList
</a>
</em>
</td>
<td>
<p>Capacity represents the actual resources of the underlying
cstor volume.</p>
</td>
</tr>
<tr>
<td>
<code>cstorVolumeRef</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#objectreference-v1-core">
Kubernetes core/v1.ObjectReference
</a>
</em>
</td>
<td>
<p>CStorVolumeRef has the information about where CstorVolumeClaim
is created from.</p>
</td>
</tr>
<tr>
<td>
<code>cstorVolumeSource</code></br>
<em>
string
</em>
</td>
<td>
<p>CStorVolumeSource contains the source volumeName@snapShotname
combaination.  This will be filled only if it is a clone creation.</p>
</td>
</tr>
<tr>
<td>
<code>provision</code></br>
<em>
<a href="#cstor.openebs.io/v1.VolumeProvision">
VolumeProvision
</a>
</em>
</td>
<td>
<p>Provision represents the initial volume configuration for the underlying
cstor volume based on the persistent volume request by user. Provision
properties are immutable</p>
</td>
</tr>
<tr>
<td>
<code>policy</code></br>
<em>
<a href="#cstor.openebs.io/v1.CStorVolumePolicySpec">
CStorVolumePolicySpec
</a>
</em>
</td>
<td>
<p>Policy contains volume specific required policies target and replicas</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>publish</code></br>
<em>
<a href="#cstor.openebs.io/v1.CStorVolumeConfigPublish">
CStorVolumeConfigPublish
</a>
</em>
</td>
<td>
<p>Publish contains info related to attachment of a volume to a node.
i.e. NodeId etc.</p>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#cstor.openebs.io/v1.CStorVolumeConfigStatus">
CStorVolumeConfigStatus
</a>
</em>
</td>
<td>
<p>Status represents the current information/status for the cstor volume
config, populated by the controller.</p>
</td>
</tr>
<tr>
<td>
<code>versionDetails</code></br>
<em>
<a href="#cstor.openebs.io/v1.VersionDetails">
VersionDetails
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="cstor.openebs.io/v1.CStorVolumePolicy">CStorVolumePolicy
</h3>
<p>
<p>CStorVolumePolicy describes a configuration required for cstor volume
resources</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
cstor.openebs.io/v1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>CStorVolumePolicy</code></td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
<a href="#cstor.openebs.io/v1.CStorVolumePolicySpec">
CStorVolumePolicySpec
</a>
</em>
</td>
<td>
<p>Spec defines a configuration info of a cstor volume required
to provisione cstor volume resources</p>
<br/>
<br/>
<table>
<tr>
<td>
<code>provision</code></br>
<em>
<a href="#cstor.openebs.io/v1.Provision">
Provision
</a>
</em>
</td>
<td>
<p>replicaAffinity is set to true then volume replica resources need to be
distributed across the pool instances</p>
</td>
</tr>
<tr>
<td>
<code>target</code></br>
<em>
<a href="#cstor.openebs.io/v1.TargetSpec">
TargetSpec
</a>
</em>
</td>
<td>
<p>TargetSpec represents configuration related to cstor target and its resources</p>
</td>
</tr>
<tr>
<td>
<code>replica</code></br>
<em>
<a href="#cstor.openebs.io/v1.ReplicaSpec">
ReplicaSpec
</a>
</em>
</td>
<td>
<p>ReplicaSpec represents configuration related to replicas resources</p>
</td>
</tr>
<tr>
<td>
<code>replicaPoolInfo</code></br>
<em>
<a href="#cstor.openebs.io/v1.ReplicaPoolInfo">
[]ReplicaPoolInfo
</a>
</em>
</td>
<td>
<p>ReplicaPoolInfo holds the pool information of volume replicas.
Ex: If volume is provisioned on which CStor pool volume replicas exist</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#cstor.openebs.io/v1.CStorVolumePolicyStatus">
CStorVolumePolicyStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="cstor.openebs.io/v1.CStorVolumeReplica">CStorVolumeReplica
</h3>
<p>
<p>CStorVolumeReplica describes a cstor volume resource created as custom resource</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
cstor.openebs.io/v1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>CStorVolumeReplica</code></td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
<a href="#cstor.openebs.io/v1.CStorVolumeReplicaSpec">
CStorVolumeReplicaSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>targetIP</code></br>
<em>
string
</em>
</td>
<td>
<p>TargetIP represents iscsi target IP through which replica cummunicates
IO workloads and other volume operations like snapshot and resize requests</p>
</td>
</tr>
<tr>
<td>
<code>capacity</code></br>
<em>
string
</em>
</td>
<td>
<p>Represents the actual capacity of the underlying volume</p>
</td>
</tr>
<tr>
<td>
<code>zvolWorkers</code></br>
<em>
string
</em>
</td>
<td>
<p>ZvolWorkers represents number of threads that executes client IOs</p>
</td>
</tr>
<tr>
<td>
<code>replicaid</code></br>
<em>
string
</em>
</td>
<td>
<p>ReplicaID is unique number to identify the replica</p>
</td>
</tr>
<tr>
<td>
<code>compression</code></br>
<em>
string
</em>
</td>
<td>
<p>Controls the compression algorithm used for this volumes
examples: on|off|gzip|gzip-N|lz4|lzjb|zle</p>
</td>
</tr>
<tr>
<td>
<code>blockSize</code></br>
<em>
uint32
</em>
</td>
<td>
<p>BlockSize is the logical block size in multiple of 512 bytes
BlockSize specifies the block size of the volume. The blocksize
cannot be changed once the volume has been written, so it should be
set at volume creation time. The default blocksize for volumes is 4 Kbytes.
Any power of 2 from 512 bytes to 128 Kbytes is valid.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#cstor.openebs.io/v1.CStorVolumeReplicaStatus">
CStorVolumeReplicaStatus
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>versionDetails</code></br>
<em>
<a href="#cstor.openebs.io/v1.VersionDetails">
VersionDetails
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="cstor.openebs.io/v1.CSPCConditionType">CSPCConditionType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#cstor.openebs.io/v1.CStorPoolClusterCondition">CStorPoolClusterCondition</a>)
</p>
<p>
</p>
<h3 id="cstor.openebs.io/v1.CSPIPredicate">CSPIPredicate
</h3>
<p>
<p>Predicate defines an abstraction to determine conditional checks against the
provided CStorPoolInstance</p>
</p>
<h3 id="cstor.openebs.io/v1.CStorPoolClusterCondition">CStorPoolClusterCondition
</h3>
<p>
(<em>Appears on:</em>
<a href="#cstor.openebs.io/v1.CStorPoolClusterStatus">CStorPoolClusterStatus</a>)
</p>
<p>
<p>CStorPoolClusterCondition describes the state of a CSPC at a certain point.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>type</code></br>
<em>
<a href="#cstor.openebs.io/v1.CSPCConditionType">
CSPCConditionType
</a>
</em>
</td>
<td>
<p>Type of CSPC condition.</p>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#conditionstatus-v1-core">
Kubernetes core/v1.ConditionStatus
</a>
</em>
</td>
<td>
<p>Status of the condition, one of True, False, Unknown.</p>
</td>
</tr>
<tr>
<td>
<code>lastUpdateTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>The last time this condition was updated.</p>
</td>
</tr>
<tr>
<td>
<code>lastTransitionTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>Last time the condition transitioned from one status to another.</p>
</td>
</tr>
<tr>
<td>
<code>reason</code></br>
<em>
string
</em>
</td>
<td>
<p>The reason for the condition&rsquo;s last transition.</p>
</td>
</tr>
<tr>
<td>
<code>message</code></br>
<em>
string
</em>
</td>
<td>
<p>A human readable message indicating details about the transition.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cstor.openebs.io/v1.CStorPoolClusterSpec">CStorPoolClusterSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#cstor.openebs.io/v1.CStorPoolCluster">CStorPoolCluster</a>)
</p>
<p>
<p>CStorPoolClusterSpec is the spec for a CStorPoolClusterSpec resource</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>pools</code></br>
<em>
<a href="#cstor.openebs.io/v1.PoolSpec">
[]PoolSpec
</a>
</em>
</td>
<td>
<p>Pools is the spec for pools for various nodes
where it should be created.</p>
</td>
</tr>
<tr>
<td>
<code>resources</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#resourcerequirements-v1-core">
Kubernetes core/v1.ResourceRequirements
</a>
</em>
</td>
<td>
<p>DefaultResources are the compute resources required by the cstor-pool
container.
If the resources at PoolConfig is not specified, this is written
to CSPI PoolConfig.</p>
</td>
</tr>
<tr>
<td>
<code>auxResources</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#resourcerequirements-v1-core">
Kubernetes core/v1.ResourceRequirements
</a>
</em>
</td>
<td>
<p>AuxResources are the compute resources required by the cstor-pool pod
side car containers.</p>
</td>
</tr>
<tr>
<td>
<code>tolerations</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#toleration-v1-core">
[]Kubernetes core/v1.Toleration
</a>
</em>
</td>
<td>
<p>Tolerations, if specified, are the pool pod&rsquo;s tolerations
If tolerations at PoolConfig is empty, this is written to
CSPI PoolConfig.</p>
</td>
</tr>
<tr>
<td>
<code>priorityClassName</code></br>
<em>
string
</em>
</td>
<td>
<p>DefaultPriorityClassName if specified applies to all the pool pods
in the pool spec if the priorityClass at the pool level is
not specified.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cstor.openebs.io/v1.CStorPoolClusterStatus">CStorPoolClusterStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#cstor.openebs.io/v1.CStorPoolCluster">CStorPoolCluster</a>)
</p>
<p>
<p>CStorPoolClusterStatus represents the latest available observations of a CSPC&rsquo;s current state.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>provisionedInstances</code></br>
<em>
int32
</em>
</td>
<td>
<p>ProvisionedInstances is the the number of CSPI present at the current state.</p>
</td>
</tr>
<tr>
<td>
<code>desiredInstances</code></br>
<em>
int32
</em>
</td>
<td>
<p>DesiredInstances is the number of CSPI(s) that should be provisioned.</p>
</td>
</tr>
<tr>
<td>
<code>healthyInstances</code></br>
<em>
int32
</em>
</td>
<td>
<p>HealthyInstances is the number of CSPI(s) that are healthy.</p>
</td>
</tr>
<tr>
<td>
<code>conditions</code></br>
<em>
<a href="#cstor.openebs.io/v1.CStorPoolClusterCondition">
[]CStorPoolClusterCondition
</a>
</em>
</td>
<td>
<p>Current state of CSPC.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cstor.openebs.io/v1.CStorPoolInstanceBlockDevice">CStorPoolInstanceBlockDevice
</h3>
<p>
(<em>Appears on:</em>
<a href="#cstor.openebs.io/v1.RaidGroup">RaidGroup</a>)
</p>
<p>
<p>CStorPoolInstanceBlockDevice contains the details of block devices that
constitutes a raid group.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>blockDeviceName</code></br>
<em>
string
</em>
</td>
<td>
<p>BlockDeviceName is the name of the block device.</p>
</td>
</tr>
<tr>
<td>
<code>capacity</code></br>
<em>
uint64
</em>
</td>
<td>
<p>Capacity is the capacity of the block device.
It is system generated</p>
</td>
</tr>
<tr>
<td>
<code>devLink</code></br>
<em>
string
</em>
</td>
<td>
<p>DevLink is the dev link for block devices</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cstor.openebs.io/v1.CStorPoolInstanceCapacity">CStorPoolInstanceCapacity
</h3>
<p>
(<em>Appears on:</em>
<a href="#cstor.openebs.io/v1.CStorPoolInstanceStatus">CStorPoolInstanceStatus</a>)
</p>
<p>
<p>CStorPoolInstanceCapacity stores the pool capacity related attributes.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>used</code></br>
<em>
k8s.io/apimachinery/pkg/api/resource.Quantity
</em>
</td>
<td>
<p>Amount of physical data (and its metadata) written to pool
after applying compression, etc..,</p>
</td>
</tr>
<tr>
<td>
<code>free</code></br>
<em>
k8s.io/apimachinery/pkg/api/resource.Quantity
</em>
</td>
<td>
<p>Amount of usable space in the pool after excluding
metadata and raid parity</p>
</td>
</tr>
<tr>
<td>
<code>total</code></br>
<em>
k8s.io/apimachinery/pkg/api/resource.Quantity
</em>
</td>
<td>
<p>Sum of usable capacity in all the data raidgroups</p>
</td>
</tr>
<tr>
<td>
<code>zfs</code></br>
<em>
<a href="#cstor.openebs.io/v1.ZFSCapacityAttributes">
ZFSCapacityAttributes
</a>
</em>
</td>
<td>
<p>ZFSCapacityAttributes contains advanced information about pool capacity details</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cstor.openebs.io/v1.CStorPoolInstanceCondition">CStorPoolInstanceCondition
</h3>
<p>
(<em>Appears on:</em>
<a href="#cstor.openebs.io/v1.CStorPoolInstanceStatus">CStorPoolInstanceStatus</a>)
</p>
<p>
<p>CSPIConditionType describes the state of a CSPI at a certain point.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>type</code></br>
<em>
<a href="#cstor.openebs.io/v1.CStorPoolInstanceConditionType">
CStorPoolInstanceConditionType
</a>
</em>
</td>
<td>
<p>Type of CSPC condition.</p>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#conditionstatus-v1-core">
Kubernetes core/v1.ConditionStatus
</a>
</em>
</td>
<td>
<p>Status of the condition, one of True, False, Unknown.</p>
</td>
</tr>
<tr>
<td>
<code>lastUpdateTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>The last time this condition was updated.</p>
</td>
</tr>
<tr>
<td>
<code>lastTransitionTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>Last time the condition transitioned from one status to another.</p>
</td>
</tr>
<tr>
<td>
<code>reason</code></br>
<em>
string
</em>
</td>
<td>
<p>The reason for the condition&rsquo;s last transition.</p>
</td>
</tr>
<tr>
<td>
<code>message</code></br>
<em>
string
</em>
</td>
<td>
<p>A human readable message indicating details about the transition.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cstor.openebs.io/v1.CStorPoolInstanceConditionType">CStorPoolInstanceConditionType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#cstor.openebs.io/v1.CStorPoolInstanceCondition">CStorPoolInstanceCondition</a>)
</p>
<p>
</p>
<h3 id="cstor.openebs.io/v1.CStorPoolInstancePhase">CStorPoolInstancePhase
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#cstor.openebs.io/v1.CStorPoolInstanceStatus">CStorPoolInstanceStatus</a>)
</p>
<p>
<p>CStorPoolInstancePhase is the phase for CStorPoolInstance resource.</p>
</p>
<h3 id="cstor.openebs.io/v1.CStorPoolInstanceSpec">CStorPoolInstanceSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#cstor.openebs.io/v1.CStorPoolInstance">CStorPoolInstance</a>)
</p>
<p>
<p>CStorPoolInstanceSpec is the spec listing fields for a CStorPoolInstance resource.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>hostName</code></br>
<em>
string
</em>
</td>
<td>
<p>HostName is the name of kubernetes node where the pool
should be created.</p>
</td>
</tr>
<tr>
<td>
<code>nodeSelector</code></br>
<em>
map[string]string
</em>
</td>
<td>
<p>NodeSelector is the labels that will be used to select
a node for pool provisioning.
Required field</p>
</td>
</tr>
<tr>
<td>
<code>poolConfig</code></br>
<em>
<a href="#cstor.openebs.io/v1.PoolConfig">
PoolConfig
</a>
</em>
</td>
<td>
<p>PoolConfig is the default pool config that applies to the
pool on node.</p>
</td>
</tr>
<tr>
<td>
<code>dataRaidGroups</code></br>
<em>
<a href="#cstor.openebs.io/v1.RaidGroup">
[]RaidGroup
</a>
</em>
</td>
<td>
<p>DataRaidGroups is the raid group configuration for the given pool.</p>
</td>
</tr>
<tr>
<td>
<code>writeCacheRaidGroups</code></br>
<em>
<a href="#cstor.openebs.io/v1.RaidGroup">
[]RaidGroup
</a>
</em>
</td>
<td>
<p>WriteCacheRaidGroups is the write cache raid group.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cstor.openebs.io/v1.CStorPoolInstanceStatus">CStorPoolInstanceStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#cstor.openebs.io/v1.CStorPoolInstance">CStorPoolInstance</a>)
</p>
<p>
<p>CStorPoolInstanceStatus is for handling status of pool.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>conditions</code></br>
<em>
<a href="#cstor.openebs.io/v1.CStorPoolInstanceCondition">
[]CStorPoolInstanceCondition
</a>
</em>
</td>
<td>
<p>Current state of CSPI with details.</p>
</td>
</tr>
<tr>
<td>
<code>phase</code></br>
<em>
<a href="#cstor.openebs.io/v1.CStorPoolInstancePhase">
CStorPoolInstancePhase
</a>
</em>
</td>
<td>
<p>The phase of a CStorPool is a simple, high-level summary of the pool state on the
node.</p>
</td>
</tr>
<tr>
<td>
<code>capacity</code></br>
<em>
<a href="#cstor.openebs.io/v1.CStorPoolInstanceCapacity">
CStorPoolInstanceCapacity
</a>
</em>
</td>
<td>
<p>Capacity describes the capacity details of a cstor pool</p>
</td>
</tr>
<tr>
<td>
<code>readOnly</code></br>
<em>
bool
</em>
</td>
<td>
<p>ReadOnly if pool is readOnly or not</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cstor.openebs.io/v1.CStorSnapshotInfo">CStorSnapshotInfo
</h3>
<p>
(<em>Appears on:</em>
<a href="#cstor.openebs.io/v1.CStorVolumeReplicaStatus">CStorVolumeReplicaStatus</a>)
</p>
<p>
<p>CStorSnapshotInfo represents the snapshot information related to particular
snapshot</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>logicalReferenced</code></br>
<em>
uint64
</em>
</td>
<td>
<p>LogicalReferenced describes the amount of space that is &ldquo;logically&rdquo;
accessible by this snapshot. This logical space ignores the
effect of the compression and copies properties, giving a quantity
closer to the amount of data that application see. It also includes
space consumed by metadata.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cstor.openebs.io/v1.CStorVolumeCondition">CStorVolumeCondition
</h3>
<p>
(<em>Appears on:</em>
<a href="#cstor.openebs.io/v1.CStorVolumeStatus">CStorVolumeStatus</a>)
</p>
<p>
<p>CStorVolumeCondition contains details about state of cstorvolume</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>type</code></br>
<em>
<a href="#cstor.openebs.io/v1.CStorVolumeConditionType">
CStorVolumeConditionType
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#cstor.openebs.io/v1.ConditionStatus">
ConditionStatus
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>lastProbeTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Last time we probed the condition.</p>
</td>
</tr>
<tr>
<td>
<code>lastTransitionTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Last time the condition transitioned from one status to another.</p>
</td>
</tr>
<tr>
<td>
<code>reason</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Unique, this should be a short, machine understandable string that gives the reason
for condition&rsquo;s last transition. If it reports &ldquo;ResizePending&rdquo; that means the underlying
cstorvolume is being resized.</p>
</td>
</tr>
<tr>
<td>
<code>message</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Human-readable message indicating details about last transition.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cstor.openebs.io/v1.CStorVolumeConditionType">CStorVolumeConditionType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#cstor.openebs.io/v1.CStorVolumeCondition">CStorVolumeCondition</a>)
</p>
<p>
<p>CStorVolumeConditionType is a valid value of CStorVolumeCondition.Type</p>
</p>
<h3 id="cstor.openebs.io/v1.CStorVolumeConfigCondition">CStorVolumeConfigCondition
</h3>
<p>
(<em>Appears on:</em>
<a href="#cstor.openebs.io/v1.CStorVolumeConfigStatus">CStorVolumeConfigStatus</a>)
</p>
<p>
<p>CStorVolumeConfigCondition contains details about state of cstor volume</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>type</code></br>
<em>
<a href="#cstor.openebs.io/v1.CStorVolumeConfigConditionType">
CStorVolumeConfigConditionType
</a>
</em>
</td>
<td>
<p>Current Condition of cstor volume config. If underlying persistent volume is being
resized then the Condition will be set to &lsquo;ResizeStarted&rsquo; etc</p>
</td>
</tr>
<tr>
<td>
<code>lastProbeTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Last time we probed the condition.</p>
</td>
</tr>
<tr>
<td>
<code>lastTransitionTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Last time the condition transitioned from one status to another.</p>
</td>
</tr>
<tr>
<td>
<code>reason</code></br>
<em>
string
</em>
</td>
<td>
<p>Reason is a brief CamelCase string that describes any failure</p>
</td>
</tr>
<tr>
<td>
<code>message</code></br>
<em>
string
</em>
</td>
<td>
<p>Human-readable message indicating details about last transition.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cstor.openebs.io/v1.CStorVolumeConfigConditionType">CStorVolumeConfigConditionType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#cstor.openebs.io/v1.CStorVolumeConfigCondition">CStorVolumeConfigCondition</a>)
</p>
<p>
<p>CStorVolumeConfigConditionType is a valid value of CstorVolumeConfigCondition.Type</p>
</p>
<h3 id="cstor.openebs.io/v1.CStorVolumeConfigPhase">CStorVolumeConfigPhase
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#cstor.openebs.io/v1.CStorVolumeConfigStatus">CStorVolumeConfigStatus</a>)
</p>
<p>
<p>CStorVolumeConfigPhase represents the current phase of CStorVolumeConfig.</p>
</p>
<h3 id="cstor.openebs.io/v1.CStorVolumeConfigPublish">CStorVolumeConfigPublish
</h3>
<p>
(<em>Appears on:</em>
<a href="#cstor.openebs.io/v1.CStorVolumeConfig">CStorVolumeConfig</a>)
</p>
<p>
<p>CStorVolumeConfigPublish contains info related to attachment of a volume to a node.
i.e. NodeId etc.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>nodeId</code></br>
<em>
string
</em>
</td>
<td>
<p>NodeID contains publish info related to attachment of a volume to a node.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cstor.openebs.io/v1.CStorVolumeConfigSpec">CStorVolumeConfigSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#cstor.openebs.io/v1.CStorVolumeConfig">CStorVolumeConfig</a>)
</p>
<p>
<p>CStorVolumeConfigSpec is the spec for a CStorVolumeConfig resource</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>capacity</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#resourcelist-v1-core">
Kubernetes core/v1.ResourceList
</a>
</em>
</td>
<td>
<p>Capacity represents the actual resources of the underlying
cstor volume.</p>
</td>
</tr>
<tr>
<td>
<code>cstorVolumeRef</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#objectreference-v1-core">
Kubernetes core/v1.ObjectReference
</a>
</em>
</td>
<td>
<p>CStorVolumeRef has the information about where CstorVolumeClaim
is created from.</p>
</td>
</tr>
<tr>
<td>
<code>cstorVolumeSource</code></br>
<em>
string
</em>
</td>
<td>
<p>CStorVolumeSource contains the source volumeName@snapShotname
combaination.  This will be filled only if it is a clone creation.</p>
</td>
</tr>
<tr>
<td>
<code>provision</code></br>
<em>
<a href="#cstor.openebs.io/v1.VolumeProvision">
VolumeProvision
</a>
</em>
</td>
<td>
<p>Provision represents the initial volume configuration for the underlying
cstor volume based on the persistent volume request by user. Provision
properties are immutable</p>
</td>
</tr>
<tr>
<td>
<code>policy</code></br>
<em>
<a href="#cstor.openebs.io/v1.CStorVolumePolicySpec">
CStorVolumePolicySpec
</a>
</em>
</td>
<td>
<p>Policy contains volume specific required policies target and replicas</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cstor.openebs.io/v1.CStorVolumeConfigStatus">CStorVolumeConfigStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#cstor.openebs.io/v1.CStorVolumeConfig">CStorVolumeConfig</a>)
</p>
<p>
<p>CStorVolumeConfigStatus is for handling status of CstorVolume Claim.
defines the observed state of CStorVolumeConfig</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>phase</code></br>
<em>
<a href="#cstor.openebs.io/v1.CStorVolumeConfigPhase">
CStorVolumeConfigPhase
</a>
</em>
</td>
<td>
<p>Phase represents the current phase of CStorVolumeConfig.</p>
</td>
</tr>
<tr>
<td>
<code>poolInfo</code></br>
<em>
[]string
</em>
</td>
<td>
<p>PoolInfo represents current pool names where volume replicas exists</p>
</td>
</tr>
<tr>
<td>
<code>capacity</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#resourcelist-v1-core">
Kubernetes core/v1.ResourceList
</a>
</em>
</td>
<td>
<p>Capacity the actual resources of the underlying volume.</p>
</td>
</tr>
<tr>
<td>
<code>condition</code></br>
<em>
<a href="#cstor.openebs.io/v1.CStorVolumeConfigCondition">
[]CStorVolumeConfigCondition
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="cstor.openebs.io/v1.CStorVolumePhase">CStorVolumePhase
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#cstor.openebs.io/v1.CStorVolumeStatus">CStorVolumeStatus</a>)
</p>
<p>
<p>CStorVolumePhase is to hold result of action.</p>
</p>
<h3 id="cstor.openebs.io/v1.CStorVolumePolicySpec">CStorVolumePolicySpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#cstor.openebs.io/v1.CStorVolumePolicy">CStorVolumePolicy</a>,
<a href="#cstor.openebs.io/v1.CStorVolumeConfigSpec">CStorVolumeConfigSpec</a>)
</p>
<p>
<p>CStorVolumePolicySpec &hellip;</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>provision</code></br>
<em>
<a href="#cstor.openebs.io/v1.Provision">
Provision
</a>
</em>
</td>
<td>
<p>replicaAffinity is set to true then volume replica resources need to be
distributed across the pool instances</p>
</td>
</tr>
<tr>
<td>
<code>target</code></br>
<em>
<a href="#cstor.openebs.io/v1.TargetSpec">
TargetSpec
</a>
</em>
</td>
<td>
<p>TargetSpec represents configuration related to cstor target and its resources</p>
</td>
</tr>
<tr>
<td>
<code>replica</code></br>
<em>
<a href="#cstor.openebs.io/v1.ReplicaSpec">
ReplicaSpec
</a>
</em>
</td>
<td>
<p>ReplicaSpec represents configuration related to replicas resources</p>
</td>
</tr>
<tr>
<td>
<code>replicaPoolInfo</code></br>
<em>
<a href="#cstor.openebs.io/v1.ReplicaPoolInfo">
[]ReplicaPoolInfo
</a>
</em>
</td>
<td>
<p>ReplicaPoolInfo holds the pool information of volume replicas.
Ex: If volume is provisioned on which CStor pool volume replicas exist</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cstor.openebs.io/v1.CStorVolumePolicyStatus">CStorVolumePolicyStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#cstor.openebs.io/v1.CStorVolumePolicy">CStorVolumePolicy</a>)
</p>
<p>
<p>CStorVolumePolicyStatus is for handling status of CstorVolumePolicy</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>phase</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="cstor.openebs.io/v1.CStorVolumeReplicaCapacityDetails">CStorVolumeReplicaCapacityDetails
</h3>
<p>
(<em>Appears on:</em>
<a href="#cstor.openebs.io/v1.CStorVolumeReplicaStatus">CStorVolumeReplicaStatus</a>)
</p>
<p>
<p>CStorVolumeReplicaCapacityDetails represents capacity information releated to volume
replica</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>total</code></br>
<em>
string
</em>
</td>
<td>
<p>The amount of space consumed by this volume replica and all its descendents</p>
</td>
</tr>
<tr>
<td>
<code>used</code></br>
<em>
string
</em>
</td>
<td>
<p>The amount of space that is &ldquo;logically&rdquo; accessible by this dataset. The logical
space ignores the effect of the compression and copies properties, giving a
quantity closer to the amount of data that applications see.  However, it does
include space consumed by metadata</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cstor.openebs.io/v1.CStorVolumeReplicaDetails">CStorVolumeReplicaDetails
</h3>
<p>
(<em>Appears on:</em>
<a href="#cstor.openebs.io/v1.CStorVolumeSpec">CStorVolumeSpec</a>,
<a href="#cstor.openebs.io/v1.CStorVolumeStatus">CStorVolumeStatus</a>)
</p>
<p>
<p>CStorVolumeReplicaDetails contains trusty replica inform which will be
updated by target</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>knownReplicas</code></br>
<em>
map[github.com/openebs/api/pkg/apis/cstor/v1.ReplicaID]string
</em>
</td>
<td>
<p>KnownReplicas represents the replicas that target can trust to read data</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cstor.openebs.io/v1.CStorVolumeReplicaPhase">CStorVolumeReplicaPhase
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#cstor.openebs.io/v1.CStorVolumeReplicaStatus">CStorVolumeReplicaStatus</a>)
</p>
<p>
<p>CStorVolumeReplicaPhase is to hold result of action.</p>
</p>
<h3 id="cstor.openebs.io/v1.CStorVolumeReplicaSpec">CStorVolumeReplicaSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#cstor.openebs.io/v1.CStorVolumeReplica">CStorVolumeReplica</a>)
</p>
<p>
<p>CStorVolumeReplicaSpec is the spec for a CStorVolumeReplica resource</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>targetIP</code></br>
<em>
string
</em>
</td>
<td>
<p>TargetIP represents iscsi target IP through which replica cummunicates
IO workloads and other volume operations like snapshot and resize requests</p>
</td>
</tr>
<tr>
<td>
<code>capacity</code></br>
<em>
string
</em>
</td>
<td>
<p>Represents the actual capacity of the underlying volume</p>
</td>
</tr>
<tr>
<td>
<code>zvolWorkers</code></br>
<em>
string
</em>
</td>
<td>
<p>ZvolWorkers represents number of threads that executes client IOs</p>
</td>
</tr>
<tr>
<td>
<code>replicaid</code></br>
<em>
string
</em>
</td>
<td>
<p>ReplicaID is unique number to identify the replica</p>
</td>
</tr>
<tr>
<td>
<code>compression</code></br>
<em>
string
</em>
</td>
<td>
<p>Controls the compression algorithm used for this volumes
examples: on|off|gzip|gzip-N|lz4|lzjb|zle</p>
</td>
</tr>
<tr>
<td>
<code>blockSize</code></br>
<em>
uint32
</em>
</td>
<td>
<p>BlockSize is the logical block size in multiple of 512 bytes
BlockSize specifies the block size of the volume. The blocksize
cannot be changed once the volume has been written, so it should be
set at volume creation time. The default blocksize for volumes is 4 Kbytes.
Any power of 2 from 512 bytes to 128 Kbytes is valid.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cstor.openebs.io/v1.CStorVolumeReplicaStatus">CStorVolumeReplicaStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#cstor.openebs.io/v1.CStorVolumeReplica">CStorVolumeReplica</a>)
</p>
<p>
<p>CStorVolumeReplicaStatus is for handling status of cvr.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>phase</code></br>
<em>
<a href="#cstor.openebs.io/v1.CStorVolumeReplicaPhase">
CStorVolumeReplicaPhase
</a>
</em>
</td>
<td>
<p>CStorVolumeReplicaPhase is to holds different phases of replica</p>
</td>
</tr>
<tr>
<td>
<code>capacity</code></br>
<em>
<a href="#cstor.openebs.io/v1.CStorVolumeReplicaCapacityDetails">
CStorVolumeReplicaCapacityDetails
</a>
</em>
</td>
<td>
<p>CStorVolumeCapacityDetails represents capacity info of replica</p>
</td>
</tr>
<tr>
<td>
<code>lastTransitionTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>LastTransitionTime refers to the time when the phase changes</p>
</td>
</tr>
<tr>
<td>
<code>lastUpdateTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>The last updated time</p>
</td>
</tr>
<tr>
<td>
<code>message</code></br>
<em>
string
</em>
</td>
<td>
<p>A human readable message indicating details about the transition.</p>
</td>
</tr>
<tr>
<td>
<code>snapshots</code></br>
<em>
<a href="#cstor.openebs.io/v1.CStorSnapshotInfo">
map[string]github.com/openebs/api/pkg/apis/cstor/v1.CStorSnapshotInfo
</a>
</em>
</td>
<td>
<p>Snapshots contains list of snapshots, and their properties,
created on CVR</p>
</td>
</tr>
<tr>
<td>
<code>pendingSnapshots</code></br>
<em>
<a href="#cstor.openebs.io/v1.CStorSnapshotInfo">
map[string]github.com/openebs/api/pkg/apis/cstor/v1.CStorSnapshotInfo
</a>
</em>
</td>
<td>
<p>PendingSnapshots contains list of pending snapshots that are not yet
available on this replica</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cstor.openebs.io/v1.CStorVolumeSpec">CStorVolumeSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#cstor.openebs.io/v1.CStorVolume">CStorVolume</a>)
</p>
<p>
<p>CStorVolumeSpec is the spec for a CStorVolume resource</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>capacity</code></br>
<em>
k8s.io/apimachinery/pkg/api/resource.Quantity
</em>
</td>
<td>
<p>Capacity represents the desired size of the underlying volume.</p>
</td>
</tr>
<tr>
<td>
<code>targetIP</code></br>
<em>
string
</em>
</td>
<td>
<p>TargetIP IP of the iSCSI target service</p>
</td>
</tr>
<tr>
<td>
<code>targetPort</code></br>
<em>
string
</em>
</td>
<td>
<p>iSCSI Target Port typically TCP ports 3260</p>
</td>
</tr>
<tr>
<td>
<code>iqn</code></br>
<em>
string
</em>
</td>
<td>
<p>Target iSCSI Qualified Name.combination of nodeBase</p>
</td>
</tr>
<tr>
<td>
<code>targetPortal</code></br>
<em>
string
</em>
</td>
<td>
<p>iSCSI Target Portal. The Portal is combination of IP:port (typically TCP ports 3260)</p>
</td>
</tr>
<tr>
<td>
<code>replicationFactor</code></br>
<em>
int
</em>
</td>
<td>
<p>ReplicationFactor represents number of volume replica created during volume
provisioning connect to the target</p>
</td>
</tr>
<tr>
<td>
<code>consistencyFactor</code></br>
<em>
int
</em>
</td>
<td>
<p>ConsistencyFactor is minimum number of volume replicas i.e. <code>RF/2 + 1</code>
has to be connected to the target for write operations. Basically more then
50% of replica has to be connected to target.</p>
</td>
</tr>
<tr>
<td>
<code>desiredReplicationFactor</code></br>
<em>
int
</em>
</td>
<td>
<p>DesiredReplicationFactor represents maximum number of replicas
that are allowed to connect to the target. Required for scale operations</p>
</td>
</tr>
<tr>
<td>
<code>replicaDetails</code></br>
<em>
<a href="#cstor.openebs.io/v1.CStorVolumeReplicaDetails">
CStorVolumeReplicaDetails
</a>
</em>
</td>
<td>
<p>ReplicaDetails refers to the trusty replica information</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cstor.openebs.io/v1.CStorVolumeStatus">CStorVolumeStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#cstor.openebs.io/v1.CStorVolume">CStorVolume</a>)
</p>
<p>
<p>CStorVolumeStatus is for handling status of cvr.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>phase</code></br>
<em>
<a href="#cstor.openebs.io/v1.CStorVolumePhase">
CStorVolumePhase
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>replicaStatuses</code></br>
<em>
<a href="#cstor.openebs.io/v1.ReplicaStatus">
[]ReplicaStatus
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>capacity</code></br>
<em>
k8s.io/apimachinery/pkg/api/resource.Quantity
</em>
</td>
<td>
<p>Represents the actual capacity of the underlying volume.</p>
</td>
</tr>
<tr>
<td>
<code>lastTransitionTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>LastTransitionTime refers to the time when the phase changes</p>
</td>
</tr>
<tr>
<td>
<code>lastUpdateTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>LastUpdateTime refers to the time when last status updated due to any
operations</p>
</td>
</tr>
<tr>
<td>
<code>message</code></br>
<em>
string
</em>
</td>
<td>
<p>A human-readable message indicating details about why the volume is in this state.</p>
</td>
</tr>
<tr>
<td>
<code>conditions</code></br>
<em>
<a href="#cstor.openebs.io/v1.CStorVolumeCondition">
[]CStorVolumeCondition
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Current Condition of cstorvolume. If underlying persistent volume is being
resized then the Condition will be set to &lsquo;ResizePending&rsquo;.</p>
</td>
</tr>
<tr>
<td>
<code>replicaDetails</code></br>
<em>
<a href="#cstor.openebs.io/v1.CStorVolumeReplicaDetails">
CStorVolumeReplicaDetails
</a>
</em>
</td>
<td>
<p>ReplicaDetails refers to the trusty replica information</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cstor.openebs.io/v1.CVRKey">CVRKey
(<code>string</code> alias)</p></h3>
<p>
<p>CVRKey represents the properties of a cstorvolumereplica</p>
</p>
<h3 id="cstor.openebs.io/v1.CVStatus">CVStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#cstor.openebs.io/v1.CVStatusResponse">CVStatusResponse</a>)
</p>
<p>
<p>CVStatus stores the status of a CstorVolume obtained from response</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>replicaStatus</code></br>
<em>
<a href="#cstor.openebs.io/v1.ReplicaStatus">
[]ReplicaStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="cstor.openebs.io/v1.CVStatusResponse">CVStatusResponse
</h3>
<p>
<p>CVStatusResponse stores the response of istgt replica command output
It may contain several volumes</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>volumeStatus</code></br>
<em>
<a href="#cstor.openebs.io/v1.CVStatus">
[]CVStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="cstor.openebs.io/v1.ConditionStatus">ConditionStatus
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#cstor.openebs.io/v1.CStorVolumeCondition">CStorVolumeCondition</a>)
</p>
<p>
<p>ConditionStatus states in which state condition is present</p>
</p>
<h3 id="cstor.openebs.io/v1.Conditions">Conditions
(<code>[]github.com/openebs/api/pkg/apis/cstor/v1.CStorVolumeCondition</code> alias)</p></h3>
<p>
<p>Conditions enables building CRUD operations on cstorvolume conditions</p>
</p>
<h3 id="cstor.openebs.io/v1.PoolConfig">PoolConfig
</h3>
<p>
(<em>Appears on:</em>
<a href="#cstor.openebs.io/v1.CStorPoolInstanceSpec">CStorPoolInstanceSpec</a>,
<a href="#cstor.openebs.io/v1.PoolSpec">PoolSpec</a>)
</p>
<p>
<p>PoolConfig is the default pool config that applies to the
pool on node.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>dataRaidGroupType</code></br>
<em>
string
</em>
</td>
<td>
<p>DataRaidGroupType is the  raid type.</p>
</td>
</tr>
<tr>
<td>
<code>writeCacheGroupType</code></br>
<em>
string
</em>
</td>
<td>
<p>WriteCacheGroupType is the write cache raid type.</p>
</td>
</tr>
<tr>
<td>
<code>thickProvision</code></br>
<em>
bool
</em>
</td>
<td>
<p>ThickProvision to enable thick provisioning
Optional &ndash; defaults to false</p>
</td>
</tr>
<tr>
<td>
<code>compression</code></br>
<em>
string
</em>
</td>
<td>
<p>Compression to enable compression
Optional &ndash; defaults to off
Possible values : lz, off</p>
</td>
</tr>
<tr>
<td>
<code>resources</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#resourcerequirements-v1-core">
Kubernetes core/v1.ResourceRequirements
</a>
</em>
</td>
<td>
<p>Resources are the compute resources required by the cstor-pool
container.</p>
</td>
</tr>
<tr>
<td>
<code>auxResources</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#resourcerequirements-v1-core">
Kubernetes core/v1.ResourceRequirements
</a>
</em>
</td>
<td>
<p>AuxResources are the compute resources required by the cstor-pool pod
side car containers.</p>
</td>
</tr>
<tr>
<td>
<code>tolerations</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#toleration-v1-core">
[]Kubernetes core/v1.Toleration
</a>
</em>
</td>
<td>
<p>Tolerations, if specified, the pool pod&rsquo;s tolerations.</p>
</td>
</tr>
<tr>
<td>
<code>priorityClassName</code></br>
<em>
string
</em>
</td>
<td>
<p>PriorityClassName if specified applies to this pool pod
If left empty, DefaultPriorityClassName is applied.
(See CStorPoolClusterSpec.DefaultPriorityClassName)
If both are empty, not priority class is applied.</p>
</td>
</tr>
<tr>
<td>
<code>roThresholdLimit</code></br>
<em>
int
</em>
</td>
<td>
<p>ROThresholdLimit is threshold(percentage base) limit
for pool read only mode. If ROThresholdLimit(%) amount
of pool storage is reached then pool will set to readonly.
NOTE:
1. If ROThresholdLimit is set to 100 then entire
pool storage will be used by default it will be set to 85%.
2. ROThresholdLimit value will be 0 &lt;= ROThresholdLimit &lt;= 100.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cstor.openebs.io/v1.PoolSpec">PoolSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#cstor.openebs.io/v1.CStorPoolClusterSpec">CStorPoolClusterSpec</a>)
</p>
<p>
<p>PoolSpec is the spec for pool on node where it should be created.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>nodeSelector</code></br>
<em>
map[string]string
</em>
</td>
<td>
<p>NodeSelector is the labels that will be used to select
a node for pool provisioning.
Required field</p>
</td>
</tr>
<tr>
<td>
<code>dataRaidGroups</code></br>
<em>
<a href="#cstor.openebs.io/v1.RaidGroup">
[]RaidGroup
</a>
</em>
</td>
<td>
<p>DataRaidGroups is the raid group configuration for the given pool.</p>
</td>
</tr>
<tr>
<td>
<code>writeCacheRaidGroups</code></br>
<em>
<a href="#cstor.openebs.io/v1.RaidGroup">
[]RaidGroup
</a>
</em>
</td>
<td>
<p>WriteCacheRaidGroups is the write cache raid group.</p>
</td>
</tr>
<tr>
<td>
<code>poolConfig</code></br>
<em>
<a href="#cstor.openebs.io/v1.PoolConfig">
PoolConfig
</a>
</em>
</td>
<td>
<p>PoolConfig is the default pool config that applies to the
pool on node.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cstor.openebs.io/v1.PoolType">PoolType
(<code>string</code> alias)</p></h3>
<p>
<p>PoolType is a label for the pool type of a cStor pool.</p>
</p>
<h3 id="cstor.openebs.io/v1.Provision">Provision
</h3>
<p>
(<em>Appears on:</em>
<a href="#cstor.openebs.io/v1.CStorVolumePolicySpec">CStorVolumePolicySpec</a>)
</p>
<p>
<p>Provision represents different provisioning policy for cstor volumes</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>replicaAffinity</code></br>
<em>
bool
</em>
</td>
<td>
<p>replicaAffinity is set to true then volume replica resources need to be
distributed across the cstor pool instances based on the given topology</p>
</td>
</tr>
<tr>
<td>
<code>blockSize</code></br>
<em>
uint32
</em>
</td>
<td>
<p>BlockSize is the logical block size in multiple of 512 bytes
BlockSize specifies the block size of the volume. The blocksize
cannot be changed once the volume has been written, so it should be
set at volume creation time. The default blocksize for volumes is 4 Kbytes.
Any power of 2 from 512 bytes to 128 Kbytes is valid.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cstor.openebs.io/v1.RaidGroup">RaidGroup
</h3>
<p>
(<em>Appears on:</em>
<a href="#cstor.openebs.io/v1.CStorPoolInstanceSpec">CStorPoolInstanceSpec</a>,
<a href="#cstor.openebs.io/v1.PoolSpec">PoolSpec</a>)
</p>
<p>
<p>RaidGroup contains the details of a raid group for the pool</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>blockDevices</code></br>
<em>
<a href="#cstor.openebs.io/v1.CStorPoolInstanceBlockDevice">
[]CStorPoolInstanceBlockDevice
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="cstor.openebs.io/v1.ReplicaID">ReplicaID
(<code>string</code> alias)</p></h3>
<p>
<p>ReplicaID is to hold replicaID information</p>
</p>
<h3 id="cstor.openebs.io/v1.ReplicaPoolInfo">ReplicaPoolInfo
</h3>
<p>
(<em>Appears on:</em>
<a href="#cstor.openebs.io/v1.CStorVolumePolicySpec">CStorVolumePolicySpec</a>)
</p>
<p>
<p>ReplicaPoolInfo represents the pool information of volume replica</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>poolName</code></br>
<em>
string
</em>
</td>
<td>
<p>PoolName represents the pool name where volume replica exists</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cstor.openebs.io/v1.ReplicaSpec">ReplicaSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#cstor.openebs.io/v1.CStorVolumePolicySpec">CStorVolumePolicySpec</a>)
</p>
<p>
<p>ReplicaSpec represents configuration related to replicas resources</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>zvolWorkers</code></br>
<em>
string
</em>
</td>
<td>
<p>IOWorkers represents number of threads that executes client IOs</p>
</td>
</tr>
<tr>
<td>
<code>compression</code></br>
<em>
string
</em>
</td>
<td>
<p>The zle compression algorithm compresses runs of zeros.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cstor.openebs.io/v1.ReplicaStatus">ReplicaStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#cstor.openebs.io/v1.CStorVolumeStatus">CStorVolumeStatus</a>,
<a href="#cstor.openebs.io/v1.CVStatus">CVStatus</a>)
</p>
<p>
<p>ReplicaStatus stores the status of replicas</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>replicaId</code></br>
<em>
string
</em>
</td>
<td>
<p>ID is replica unique identifier</p>
</td>
</tr>
<tr>
<td>
<code>mode</code></br>
<em>
string
</em>
</td>
<td>
<p>Mode represents replica status i.e. Healthy, Degraded</p>
</td>
</tr>
<tr>
<td>
<code>checkpointedIOSeq</code></br>
<em>
string
</em>
</td>
<td>
<p>Represents IO number of replica persisted on the disk</p>
</td>
</tr>
<tr>
<td>
<code>inflightRead</code></br>
<em>
string
</em>
</td>
<td>
<p>Ongoing reads I/O from target to replica</p>
</td>
</tr>
<tr>
<td>
<code>inflightWrite</code></br>
<em>
string
</em>
</td>
<td>
<p>ongoing writes I/O from target to replica</p>
</td>
</tr>
<tr>
<td>
<code>inflightSync</code></br>
<em>
string
</em>
</td>
<td>
<p>Ongoing sync I/O from target to replica</p>
</td>
</tr>
<tr>
<td>
<code>upTime</code></br>
<em>
int
</em>
</td>
<td>
<p>time since the replica connected to target</p>
</td>
</tr>
<tr>
<td>
<code>quorum</code></br>
<em>
string
</em>
</td>
<td>
<p>Quorum indicates whether data wrtitten to the replica
is lost or exists.
&ldquo;0&rdquo; means: data has been lost( might be ephimeral case)
and will recostruct data from other Healthy replicas in a write-only
mode
1 means: written data is exists on replica</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cstor.openebs.io/v1.TargetSpec">TargetSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#cstor.openebs.io/v1.CStorVolumePolicySpec">CStorVolumePolicySpec</a>)
</p>
<p>
<p>TargetSpec represents configuration related to cstor target and its resources</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>queueDepth</code></br>
<em>
string
</em>
</td>
<td>
<p>QueueDepth sets the queue size at iSCSI target which limits the
ongoing IO count from client</p>
</td>
</tr>
<tr>
<td>
<code>luWorkers</code></br>
<em>
int64
</em>
</td>
<td>
<p>IOWorkers sets the number of threads that are working on above queue</p>
</td>
</tr>
<tr>
<td>
<code>monitor</code></br>
<em>
bool
</em>
</td>
<td>
<p>Monitor enables or disables the target exporter sidecar</p>
</td>
</tr>
<tr>
<td>
<code>replicationFactor</code></br>
<em>
int64
</em>
</td>
<td>
<p>ReplicationFactor represents maximum number of replicas
that are allowed to connect to the target</p>
</td>
</tr>
<tr>
<td>
<code>resources</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#resourcerequirements-v1-core">
Kubernetes core/v1.ResourceRequirements
</a>
</em>
</td>
<td>
<p>Resources are the compute resources required by the cstor-target
container.</p>
</td>
</tr>
<tr>
<td>
<code>auxResources</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#resourcerequirements-v1-core">
Kubernetes core/v1.ResourceRequirements
</a>
</em>
</td>
<td>
<p>AuxResources are the compute resources required by the cstor-target pod
side car containers.</p>
</td>
</tr>
<tr>
<td>
<code>tolerations</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#toleration-v1-core">
[]Kubernetes core/v1.Toleration
</a>
</em>
</td>
<td>
<p>Tolerations, if specified, are the target pod&rsquo;s tolerations</p>
</td>
</tr>
<tr>
<td>
<code>affinity</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#podaffinity-v1-core">
Kubernetes core/v1.PodAffinity
</a>
</em>
</td>
<td>
<p>PodAffinity if specified, are the target pod&rsquo;s affinities</p>
</td>
</tr>
<tr>
<td>
<code>nodeSelector</code></br>
<em>
map[string]string
</em>
</td>
<td>
<p>NodeSelector is the labels that will be used to select
a node for target pod scheduling
Required field</p>
</td>
</tr>
<tr>
<td>
<code>priorityClassName</code></br>
<em>
string
</em>
</td>
<td>
<p>PriorityClassName if specified applies to this target pod
If left empty, no priority class is applied.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cstor.openebs.io/v1.VersionDetails">VersionDetails
</h3>
<p>
(<em>Appears on:</em>
<a href="#cstor.openebs.io/v1.CStorPoolCluster">CStorPoolCluster</a>,
<a href="#cstor.openebs.io/v1.CStorPoolInstance">CStorPoolInstance</a>,
<a href="#cstor.openebs.io/v1.CStorVolume">CStorVolume</a>,
<a href="#cstor.openebs.io/v1.CStorVolumeConfig">CStorVolumeConfig</a>,
<a href="#cstor.openebs.io/v1.CStorVolumeReplica">CStorVolumeReplica</a>)
</p>
<p>
<p>VersionDetails provides the details for upgrade</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>autoUpgrade</code></br>
<em>
bool
</em>
</td>
<td>
<p>If AutoUpgrade is set to true then the resource is
upgraded automatically without any manual steps</p>
</td>
</tr>
<tr>
<td>
<code>desired</code></br>
<em>
string
</em>
</td>
<td>
<p>Desired is the version that we want to
upgrade or the control plane version</p>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#cstor.openebs.io/v1.VersionStatus">
VersionStatus
</a>
</em>
</td>
<td>
<p>Status gives the status of reconciliation triggered
when the desired and current version are not same</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cstor.openebs.io/v1.VersionState">VersionState
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#cstor.openebs.io/v1.VersionStatus">VersionStatus</a>)
</p>
<p>
<p>VersionState is the state of reconciliation</p>
</p>
<h3 id="cstor.openebs.io/v1.VersionStatus">VersionStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#cstor.openebs.io/v1.VersionDetails">VersionDetails</a>)
</p>
<p>
<p>VersionStatus is the status of the reconciliation of versions</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>dependentsUpgraded</code></br>
<em>
bool
</em>
</td>
<td>
<p>DependentsUpgraded gives the details whether all children
of a resource are upgraded to desired version or not</p>
</td>
</tr>
<tr>
<td>
<code>current</code></br>
<em>
string
</em>
</td>
<td>
<p>Current is the version of resource</p>
</td>
</tr>
<tr>
<td>
<code>state</code></br>
<em>
<a href="#cstor.openebs.io/v1.VersionState">
VersionState
</a>
</em>
</td>
<td>
<p>State is the state of reconciliation</p>
</td>
</tr>
<tr>
<td>
<code>message</code></br>
<em>
string
</em>
</td>
<td>
<p>Message is a human readable message if some error occurs</p>
</td>
</tr>
<tr>
<td>
<code>reason</code></br>
<em>
string
</em>
</td>
<td>
<p>Reason is the actual reason for the error state</p>
</td>
</tr>
<tr>
<td>
<code>lastUpdateTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>LastUpdateTime is the time the status was last  updated</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cstor.openebs.io/v1.VolumeProvision">VolumeProvision
</h3>
<p>
(<em>Appears on:</em>
<a href="#cstor.openebs.io/v1.CStorVolumeConfigSpec">CStorVolumeConfigSpec</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>capacity</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#resourcelist-v1-core">
Kubernetes core/v1.ResourceList
</a>
</em>
</td>
<td>
<p>Capacity represents initial capacity of volume replica required during
volume clone operations to maintain some metadata info related to child
resources like snapshot, cloned volumes.</p>
</td>
</tr>
<tr>
<td>
<code>replicaCount</code></br>
<em>
int
</em>
</td>
<td>
<p>ReplicaCount represents initial cstor volume replica count, its will not
be updated later on based on scale up/down operations, only readonly
operations and validations.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="cstor.openebs.io/v1.ZFSCapacityAttributes">ZFSCapacityAttributes
</h3>
<p>
(<em>Appears on:</em>
<a href="#cstor.openebs.io/v1.CStorPoolInstanceCapacity">CStorPoolInstanceCapacity</a>)
</p>
<p>
<p>ZFSCapacityAttributes stores the advanced information about pool capacity related
attributes</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>logicalUsed</code></br>
<em>
k8s.io/apimachinery/pkg/api/resource.Quantity
</em>
</td>
<td>
<p>LogicalUsed is the amount of space that is &ldquo;logically&rdquo; consumed
by this pool and all its descendents. The logical space ignores
the effect of the compression and copies properties, giving a
quantity closer to the amount of data that applications see.
However, it does include space consumed by metadata.</p>
</td>
</tr>
</tbody>
</table>
<hr/>
<p><em>
Generated with <code>gen-crd-api-reference-docs</code>
on git commit <code>81e1720</code>.
</em></p>
