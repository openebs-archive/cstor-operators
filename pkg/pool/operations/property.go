package v1alpha2

import (
	"sync"

	cstor "github.com/openebs/api/v2/pkg/apis/cstor/v1"
	zcmd "github.com/openebs/cstor-operators/pkg/zcmd"
	"github.com/pkg/errors"
)

// properties will hold required properties of zfs
type properties struct {
	fsProperties map[string]string
	// We can also add zpoolProperties if required
}

// dataset can hold fields relavant to pool dataset
type dataset struct {
	datasetsToProperties map[string]properties
	mutex                sync.Mutex
}

var (
	ds = dataset{
		datasetsToProperties: make(map[string]properties),
	}
)

// SetPoolFSPropertiesIfNot will set the given properties for pool dataset using zfs cmd utility
// only if doesn't match to existing value
func (oc *OperationsConfig) SetPoolFSPropertiesIfNot(datasetName string, desiredFSProperties map[string]string) error {
	// unsetPropValues will contain zvol propeties and their values only if they
	// doesn't exist in-memory (Or) existing value doesn't match to desired one
	var unsetPropValues map[string]string

	// To synchronize if CSPI controller launches with multiple workers
	ds.mutex.Lock()
	defer ds.mutex.Unlock()

	dsProperties := ds.datasetsToProperties[datasetName]
	if dsProperties.fsProperties == nil {
		dsProperties.fsProperties = make(map[string]string)
	}
	unsetPropValues = make(map[string]string)

	for property, desiredValue := range desiredFSProperties {
		existingValue, isPropExist := dsProperties.fsProperties[property]
		if isPropExist && existingValue == desiredValue {
			continue
		}
		// Read the property from cstor-pool container
		existingPropValue, err := oc.GetVolumePropertyValue(datasetName, property)
		if err != nil {
			return errors.Wrapf(err, "failed to get value of filesystem property %s of pool %s", property, datasetName)
		}
		if existingPropValue == desiredValue {
			dsProperties.fsProperties[property] = desiredValue
			continue
		}
		// Add property and their value which needs to configure
		unsetPropValues[property] = desiredValue
	}

	// If there are no properties to set retrun from here
	if len(unsetPropValues) == 0 {
		ds.datasetsToProperties[datasetName] = dsProperties
		return nil
	}

	zfsCommand := zcmd.NewVolumeSetProperty().
		WithDataset(datasetName).
		WithExecutor(oc.zcmdExecutor)

	for key, value := range unsetPropValues {
		zfsCommand.WithProperty(key, value)
	}
	ret, err := zfsCommand.Execute()
	if err != nil {
		return errors.Wrapf(err, "failed to set property values %v output: %s", unsetPropValues, string(ret))
	}

	// store unset properties in-memory so that in subsequent reconciliation
	for key, value := range unsetPropValues {
		dsProperties.fsProperties[key] = value
	}
	ds.datasetsToProperties[datasetName] = dsProperties

	return nil
}

// SetPoolProperties will configure required pool properties
// NOTE: Advantage of using SetPoolProperties is it avoids calls to cstor-pool
//		 executing zpool/zfs commands(most of them) will read from disk
func (oc *OperationsConfig) SetPoolProperties(cspi *cstor.CStorPoolInstance) error {
	fsProperties := map[string]string{
		"canmount": "off"}

	// default compression type is lz4
	compressionType := "lz4"
	if cspi.Spec.PoolConfig.Compression != "" {
		compressionType = cspi.Spec.PoolConfig.Compression
	}
	fsProperties["compression"] = compressionType

	err := oc.SetPoolFSPropertiesIfNot(PoolName(), fsProperties)
	if err != nil {
		return err
	}
	return nil
}
