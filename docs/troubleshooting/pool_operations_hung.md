# Pool Operation Hung Due to Bad Disk

As of now, cStor scans all the devices on the node when it tries to import the pool in case of a pool manager pod
restart. If any of the devices(even the devices that are not part of the cStor pool) are bad and is not responding the 
command issued by cStor keeps on waiting and is stuck. As a result of this, pool manager pod is not able to issue any 
more command in order to reconcile the state of cStor pools or even perform the IO for the volumes that are placed on 
that particular pool. 

A fix in this is being worked upong and more details can be found in the following link:
https://github.com/openebs/openebs/pull/2935

# How to recover from this situation?

Now there could be two different situations :

1. The device that has gone bad is actually a part of the cStor pool on the node. If this is the case
then see [Block Device Replacement](../tutorial/cspc/tuning/tune.md) in this tutorial. This tutorial is for
`mirror` RAID but similar approach can be followed for other RAID configurations like `raidz1` and `raidz2`. 

**Note:** Block device replacement is not supported for `stripe` raid configuration. 

2. The device that has gone bad is not part of the cStor pool on the node. in this case, perform the following steps
to recover from this situation:
- Remove the bad disk from the node.
- Restart the pool manager pod.
