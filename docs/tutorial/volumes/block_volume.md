# CStor CSI RAW Block Volume:

When using a raw block volume in your Pods, you must specify a VolumeDevice attribute
in the Container section of the PodSpec rather than a VolumeMount. VolumeDevices
have devicePaths instead of mountPaths, and inside the container, applications 
will see a device at that path instead of a mounted file system.

A block volume will appear as a block device inside the container,and allows 
low-level access to the storage without intermediate layers, as with file-system
volumes.There are advantages of raw disk partitions for example:

- Block devices that are actually SCSI disks support sending SCSI commands to the
  device using Linux ioctls.
- Faster I/O without UNIX file system overhead, and more synchronous I/O without
  UNIX file system buffering etc.

### How to use block volume

There is a number of use cases where a raw block device will be useful. For example,
A user can use a raw block device for database applications such as MySQL to read 
data from and write the results to a disk that has a formatted filesystem to be 
displayed via the nginx web server.

The following sections describe some sample volume specifications and steps to dynamically 
provision a raw block volume and attach it to an application pod.

### Creating a new raw block PVC
1. Raw Block volume:

```yaml
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: block-claim
spec:
  volumeMode: Block
  storageClassName: cstor-disk-stripe
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 5Gi

```

2. Filesystem based volume:

```yaml
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: fs-claim
spec:
  storageClassName: cstor-disk-stripe
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 5Gi
```


### Using a raw Block PVC in POD

Here, the user creates an application pod/deployment, having 2 containers consume both block &
filesystem volumes. We have to choose devicePath for the block device inside the
Mysql container rather than the mountPath for the file system. Nginx container
we have mounted the filesystem based volume.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    run: mysql
  name: mysql
spec:
  replicas: 1
  selector:
    matchLabels:
      run: mysql
  strategy: {}
  template:
    metadata:
      labels:
        run: mysql
    spec:
      containers:
      - image: mysql
        imagePullPolicy: IfNotPresent
        name: mysql
        volumeDevices:
          - name: my-db-data
            devicePath: /dev/block
        env:
          - name: MYSQL_ROOT_PASSWORD
            value: test
      - image: nginx
        imagePullPolicy: IfNotPresent
        name: nginx
        ports:
          - containerPort: 80
        volumeMounts:
          - name: my-nginx-data
            mountPath: /usr/share/nginx/html
            readOnly: false
      volumes:
        - name: my-db-data
          persistentVolumeClaim:
            claimName: block-claim
        - name: my-nginx-data
          persistentVolumeClaim:
            claimName: fs-claim
```

