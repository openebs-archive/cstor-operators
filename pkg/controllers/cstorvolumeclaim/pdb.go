package cstorvolumeclaim

import (
	"fmt"

	"github.com/openebs/api/pkg/apis/types"
	policy "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// GetPDBPoolLabels returns the pool labels from poolNames
func GetPDBPoolLabels(poolNames []string) map[string]string {
	pdbLabels := map[string]string{}
	for _, poolName := range poolNames {
		key := fmt.Sprintf("openebs.io/%s", poolName)
		pdbLabels[key] = "true"
	}
	return pdbLabels
}

// GetPDBLabels returns the labels required for building PDB based on arguments
func GetPDBLabels(poolNames []string, cspcName string) map[string]string {
	pdbLabels := GetPDBPoolLabels(poolNames)
	pdbLabels[string(types.CStorPoolClusterLabelKey)] = cspcName
	return pdbLabels
}

// GetPDBLabelSelector returns the labelSelector to list the PDB
func GetPDBLabelSelector(poolNames []string) string {
	var labelSelector string
	pdbLabels := GetPDBPoolLabels(poolNames)

	for key, value := range pdbLabels {
		labelSelector = labelSelector + key + "=" + value + ","
	}
	return labelSelector[:len(labelSelector)-1]
}

// createPDB creates PDB for cStorVolumes based on arguments
func (c *CVCController) createPDB(poolNames []string, cspcName string) (*policy.PodDisruptionBudget, error) {
	// Calculate minAvailable value from cStorVolume replica count
	//minAvailable := (cvObj.Spec.ReplicationFactor >> 1) + 1
	maxUnavailableIntStr := intstr.FromInt(1)

	//build podDisruptionBudget for volume
	pdbObj := &policy.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: cspcName,
			Labels:       GetPDBLabels(poolNames, cspcName),
		},
		Spec: policy.PodDisruptionBudgetSpec{
			MaxUnavailable: &maxUnavailableIntStr,
			Selector:       getPDBSelector(poolNames),
		},
	}
	// Create podDisruptionBudget
	return c.kubeclientset.PolicyV1beta1().PodDisruptionBudgets(openebsNamespace).
		Create(pdbObj)
}

// getPDBSelector returns PDB label selector from list of pools
func getPDBSelector(pools []string) *metav1.LabelSelector {
	selectorRequirements := []metav1.LabelSelectorRequirement{}
	selectorRequirements = append(
		selectorRequirements,
		metav1.LabelSelectorRequirement{
			Key:      string(types.CStorPoolInstanceLabelKey),
			Operator: metav1.LabelSelectorOpIn,
			Values:   pools,
		})
	return &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"app": "cstor-pool",
		},
		MatchExpressions: selectorRequirements,
	}
}
