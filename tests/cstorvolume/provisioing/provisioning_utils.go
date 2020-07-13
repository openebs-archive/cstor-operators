package provisioning

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	cstorapis "github.com/openebs/api/pkg/apis/cstor/v1"
	"github.com/openebs/cstor-operators/tests/pkg/cspc/cspcspecbuilder"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	cstorCSIProvisionerName = "cstor.csi.openebs.io"
)

// createStorageClass in etcd
func createStorageClass(scName string, parameters map[string]string) *storagev1.StorageClass {
	// Building StorageClass
	var err error
	var isVolumeExpansionAllowed bool
	sc := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: scName,
		},
		Provisioner:          cstorCSIProvisionerName,
		AllowVolumeExpansion: &isVolumeExpansionAllowed,
		Parameters:           parameters,
	}

	By(fmt.Sprintf("Creating %s StorageClass", sc.Name))

	sc, err = cstorsuite.client.KubeClientSet.StorageV1().StorageClasses().Create(sc)
	Expect(err).To(BeNil())
	return sc
}

func createPersistentVolumeClaim(pvcName, pvcNamespace, scName string) *corev1.PersistentVolumeClaim {
	var err error
	resCapacity, err := resource.ParseQuantity("5G")
	Expect(err).To(BeNil())

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: pvcNamespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: &scName,
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceName(corev1.ResourceStorage): resCapacity,
				},
			},
		},
	}
	
	pvc, err = cstorsuite.client.KubeClientSet.
		CoreV1().
		PersistentVolumeClaims(pvcNamespace).
		Create(pvc)
	Expect(err).To(BeNil())
	
	return pvc
}

// ProvisionCSPC will create CSPC based cStor pools
func ProvisionCSPC(cspcName, namespace, poolType string, bdCount int) {
	specBuilder = cspcspecbuilder.
		NewCSPCSpecBuilder(cstorsuite.CSPCCache, cstorsuite.infra)

	cspc = specBuilder.BuildCSPC(cspcName, namespace, poolType, bdCount, cstorsuite.infra.NodeCount).GetCSPCSpec()
	_, err := cstorsuite.
		client.
		OpenEBSClientSet.
		CstorV1().
		CStorPoolClusters(cspc.Namespace).
		Create(cspc)
	Expect(err).To(BeNil())

	gotHealthyCSPiCount := cstorsuite.
		client.
		GetOnlineCSPICountEventually(cspc.Name, cspc.Namespace, cstorsuite.infra.NodeCount)
	Expect(gotHealthyCSPiCount).To(BeNumerically("==", int32(cstorsuite.infra.NodeCount)))
	
	// TODO: Uncoment the below code
	// cvcSpecBuilder = cvcspecbuilder.NewCVCSpecBuilder(cstorsuite.infra, []string{})

}

// DeProvisionCSPC will de-provision CSPC based cStor pools
func DeProvisionCSPC(cspc *cstorapis.CStorPoolCluster) {
	err := cstorsuite.
		client.
		OpenEBSClientSet.
		CstorV1().
		CStorPoolClusters(cspc.Namespace).
		Delete(cspc.Name, &metav1.DeleteOptions{})
	Expect(err).To(BeNil())

	specBuilder.ResetCSPCSpecData()

	gotCSPICount := cstorsuite.
		client.
		GetCSPICountEventually(cspc.Name, cspc.Namespace, 0)
	Expect(gotCSPICount).To(BeNumerically("==", 0))
}
