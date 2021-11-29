/*
Copyright 2020 The OpenEBS Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package webhook

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/openebs/api/v3/pkg/apis/types"
	"github.com/openebs/api/v3/pkg/util"
	"github.com/openebs/cstor-operators/pkg/version"
	"github.com/pkg/errors"
	admissionregistration "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

const (
	validatorServiceName = "openebs-cstor-admission-server"
	validatorWebhook     = "openebs-cstor-validation-webhook"
	validatorSecret      = "openebs-cstor-admission-secret"
	webhookHandlerName   = "admission-webhook.cstor.openebs.io"
	validationPath       = "/validate"
	validationPort       = 8443
	webhookLabel         = "openebs.io/component-name" + "=" + "cstor-admission-webhook"
	webhooksvcLabel      = "openebs.io/component-name" + "=" + "cstor-admission-webhook"
	// AdmissionNameEnvVar is the constant for env variable ADMISSION_WEBHOOK_NAME
	// which is the name of the current admission webhook
	AdmissionNameEnvVar = "ADMISSION_WEBHOOK_NAME"
	appCrt              = "app.crt"
	appKey              = "app.pem"
	rootCrt             = "ca.crt"
)

type client struct {
	// kubeClient is a standard kubernetes clientset
	kubeClient kubernetes.Interface
}

type transformSvcFunc func(*corev1.Service)
type transformSecretFunc func(*corev1.Secret)
type transformConfigFunc func(*admissionregistration.ValidatingWebhookConfiguration)

var (
	// TimeoutSeconds specifies the timeout for this webhook. After the timeout passes,
	// the webhook call will be ignored or the API call will fail based on the
	// failure policy.
	// The timeout value must be between 1 and 30 seconds.
	five = int32(5)
	// Ignore means that an error calling the webhook is ignored.
	Ignore = admissionregistration.Ignore
	// Fail means that an error calling the webhook causes the admission to fail.
	Fail = admissionregistration.Fail
	// SideEffectClassNone means that calling the webhook will have no side effects.
	SideEffectClassNone = admissionregistration.SideEffectClassNone
	// WebhookFailurePolicye represents failure policy env name to make it configurable
	// via ENV
	WebhookFailurePolicy = "ADMISSION_WEBHOOK_FAILURE_POLICY"
	// transformation function lists to upgrade webhook resources
	transformSecret = []transformSecretFunc{}
	transformSvc    = []transformSvcFunc{}
	transformConfig = []transformConfigFunc{
		addNSWithDeleteRule,
	}
	cvcRuleWithOperations = admissionregistration.RuleWithOperations{
		Operations: []admissionregistration.OperationType{
			admissionregistration.Update,
		},
		Rule: admissionregistration.Rule{
			APIGroups:   []string{"cstor.openebs.io"},
			APIVersions: []string{"v1"},
			Resources:   []string{"cstorvolumeconfigs"},
		},
	}
	nsRuleWithOperations = admissionregistration.RuleWithOperations{
		Operations: []admissionregistration.OperationType{
			admissionregistration.Delete,
		},
		Rule: admissionregistration.Rule{
			APIGroups:   []string{"*"},
			APIVersions: []string{"*"},
			Resources:   []string{"namespaces"},
		},
	}
)

// createWebhookService creates our webhook Service resource if it does not
// exist.
func (c *client) createWebhookService(
	ownerReference metav1.OwnerReference,
	serviceName string,
	namespace string,
) error {

	_, err := c.kubeClient.CoreV1().Services(namespace).
		Get(context.TODO(), serviceName, metav1.GetOptions{})

	if err == nil {
		return nil
	}

	// error other than 'not found', return err
	if !k8serror.IsNotFound(err) {
		return errors.Wrapf(
			err,
			"failed to get webhook service {%v}",
			serviceName,
		)
	}

	// create service resource that refers to admission server pod
	serviceLabels := map[string]string{"app": "cstor-admission-webhook"}
	svcObj := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      serviceName,
			Labels: map[string]string{
				"app":                                "cstor-admission-webhook",
				"openebs.io/component-name":          "cstor-admission-webhook",
				string(types.OpenEBSVersionLabelKey): version.GetVersion(),
			},
			OwnerReferences: []metav1.OwnerReference{ownerReference},
		},
		Spec: corev1.ServiceSpec{
			Selector: serviceLabels,
			Ports: []corev1.ServicePort{
				{
					Protocol:   "TCP",
					Port:       443,
					TargetPort: intstr.FromInt(validationPort),
				},
			},
		},
	}
	_, err = c.kubeClient.CoreV1().Services(namespace).
		Create(context.TODO(), svcObj, metav1.CreateOptions{})
	return err
}

// createAdmissionValidatingConfig creates our ValidatingWebhookConfiguration resource
// if it does not exist.
func (c *client) createAdmissionValidatingConfig(
	ownerReference metav1.OwnerReference,
	validatorWebhook string,
	namespace string,
	serviceName string,
	signingCert []byte,
) error {

	_, err := GetValidatorWebhook(validatorWebhook, c.kubeClient)
	// validator object already present, no need to do anything
	if err == nil {
		return nil
	}

	// error other than 'not found', return err
	if !k8serror.IsNotFound(err) {
		return errors.Wrapf(
			err,
			"failed to get validating WebhookConfiguration for {%v}",
			validatorWebhook,
		)
	}

	// sideEffectClass := admissionregistration.SideEffectClassNoneOnDryRun

	webhookHandler := admissionregistration.ValidatingWebhook{
		Name: webhookHandlerName,
		Rules: []admissionregistration.RuleWithOperations{{
			Operations: []admissionregistration.OperationType{
				admissionregistration.Create,
				admissionregistration.Delete,
			},
			Rule: admissionregistration.Rule{
				APIGroups:   []string{"*"},
				APIVersions: []string{"*"},
				Resources:   []string{"persistentvolumeclaims"},
			},
		},
			{
				Operations: []admissionregistration.OperationType{
					admissionregistration.Create,
					admissionregistration.Update,
					admissionregistration.Delete,
				},
				Rule: admissionregistration.Rule{
					APIGroups:   []string{"cstor.openebs.io"},
					APIVersions: []string{"v1"},
					Resources:   []string{"cstorpoolclusters"},
				},
			},
			cvcRuleWithOperations,
			nsRuleWithOperations,
		},
		ClientConfig: admissionregistration.WebhookClientConfig{
			Service: &admissionregistration.ServiceReference{
				Namespace: namespace,
				Name:      serviceName,
				Path:      StrPtr(validationPath),
			},
			CABundle: signingCert,
		},
		SideEffects:             &SideEffectClassNone,
		AdmissionReviewVersions: []string{"v1"},
		TimeoutSeconds:          &five,
		FailurePolicy:           failurePolicy(),
	}

	validator := &admissionregistration.ValidatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{
			Kind:       "validatingWebhookConfiguration",
			APIVersion: "admissionregistration.k8s.io/admissionregistration",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: validatorWebhook,
			Labels: map[string]string{
				"app":                                "cstor-admission-webhook",
				"openebs.io/component-name":          "cstor-admission-webhook",
				string(types.OpenEBSVersionLabelKey): version.GetVersion(),
			},
			OwnerReferences: []metav1.OwnerReference{ownerReference},
		},
		Webhooks: []admissionregistration.ValidatingWebhook{webhookHandler},
	}

	_, err = c.kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().
		Create(context.TODO(), validator, metav1.CreateOptions{})

	return err
}

// createCertsSecret creates a self-signed certificate and stores it as a
// secret resource in Kubernetes.
func (c *client) createCertsSecret(
	ownerReference metav1.OwnerReference,
	secretName string,
	serviceName string,
	namespace string,
) (*corev1.Secret, error) {

	// Create a signing certificate
	caKeyPair, err := NewCA(fmt.Sprintf("%s-ca", serviceName))
	if err != nil {
		return nil, fmt.Errorf("failed to create root-ca: %v", err)
	}

	// Create app certs signed through the certificate created above
	apiServerKeyPair, err := NewServerKeyPair(
		caKeyPair,
		strings.Join([]string{serviceName, namespace, "svc"}, "."),
		serviceName,
		namespace,
		"cluster.local",
		[]string{},
		[]string{},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create server key pair: %v", err)
	}

	// create an opaque secret resource with certificate(s) created above
	secretObj := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
			Labels: map[string]string{
				"app":                                "cstor-admission-webhook",
				"openebs.io/component-name":          "cstor-admission-webhook",
				string(types.OpenEBSVersionLabelKey): version.GetVersion(),
			},
			OwnerReferences: []metav1.OwnerReference{ownerReference},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			appCrt:  EncodeCertPEM(apiServerKeyPair.Cert),
			appKey:  EncodePrivateKeyPEM(apiServerKeyPair.Key),
			rootCrt: EncodeCertPEM(caKeyPair.Cert),
		},
	}

	return c.kubeClient.CoreV1().Secrets(namespace).Create(context.TODO(), secretObj, metav1.CreateOptions{})
}

// GetValidatorWebhook fetches the webhook validator resource in
// Openebs namespace.
func GetValidatorWebhook(
	validator string, kubeClient kubernetes.Interface,
) (*admissionregistration.ValidatingWebhookConfiguration, error) {

	return kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(context.TODO(), validator, metav1.GetOptions{})
}

// StrPtr convert a string to a pointer
func StrPtr(s string) *string {
	return &s
}

// InitValidationServer creates secret, service and admission validation k8s
// resources. All these resources are created in the same namespace where
// openebs components is running.
func InitValidationServer(
	ownerReference metav1.OwnerReference,
	k kubernetes.Interface,
) error {

	// Fetch our namespace
	openebsNamespace, err := getOpenebsNamespace()
	if err != nil {
		return err
	}

	c := &client{
		kubeClient: k,
	}

	err = c.preUpgrade(openebsNamespace)
	if err != nil {
		return err
	}

	// Check to see if webhook secret is already present
	certSecret, err := GetSecret(openebsNamespace, validatorSecret, c.kubeClient)
	if err != nil {
		if k8serror.IsNotFound(err) {
			// Secret not found, create certs and the secret object
			certSecret, err = c.createCertsSecret(
				ownerReference,
				validatorSecret,
				validatorServiceName,
				openebsNamespace,
			)
			if err != nil {
				return fmt.Errorf(
					"failed to create secret(%s) resource %v",
					validatorSecret,
					err,
				)
			}
		} else {
			// Unable to read secret object
			return fmt.Errorf(
				"unable to read secret object %s: %v",
				validatorSecret,
				err,
			)
		}
	}

	signingCertBytes, ok := certSecret.Data[rootCrt]
	if !ok {
		return fmt.Errorf(
			"%s value not found in %s secret",
			rootCrt,
			validatorSecret,
		)
	}

	serviceErr := c.createWebhookService(
		ownerReference,
		validatorServiceName,
		openebsNamespace,
	)
	if serviceErr != nil {
		return fmt.Errorf(
			"failed to create Service{%s}: %v",
			validatorServiceName,
			serviceErr,
		)
	}

	validatorErr := c.createAdmissionValidatingConfig(
		ownerReference,
		validatorWebhook,
		openebsNamespace,
		validatorServiceName,
		signingCertBytes,
	)
	if validatorErr != nil {
		return fmt.Errorf(
			"failed to create validator{%s}: %v",
			validatorWebhook,
			validatorErr,
		)
	}

	return nil
}

// GetSecret fetches the secret resource in the given namespace.
func GetSecret(
	namespace string,
	secretName string,
	kubeClient kubernetes.Interface,
) (*corev1.Secret, error) {

	return kubeClient.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
}

// getOpenebsNamespace gets the namespace OPENEBS_NAMESPACE env value which is
// set by the downward API where admission server has been deployed
func getOpenebsNamespace() (string, error) {

	ns, found := os.LookupEnv(util.OpenEBSNamespace)
	if !found {
		return "", fmt.Errorf("%s must be set", util.OpenEBSNamespace)
	}
	return ns, nil
}

func addNSWithDeleteRule(config *admissionregistration.ValidatingWebhookConfiguration) {
	if IsCurrentLessThanNewVersion(config.Labels[string(types.OpenEBSVersionLabelKey)], "2.5.0") {
		config.Webhooks[0].Rules = append(config.Webhooks[0].Rules, nsRuleWithOperations)
	}
}

// GetAdmissionName return the admission server name
func GetAdmissionName() (string, error) {
	admissionName, found := os.LookupEnv(AdmissionNameEnvVar)
	if !found {
		return "", fmt.Errorf("%s must be set", AdmissionNameEnvVar)
	}
	if len(admissionName) == 0 {
		return "", fmt.Errorf("%s must not be empty", AdmissionNameEnvVar)
	}
	return admissionName, nil
}

// GetAdmissionReference is a utility function to fetch a reference
// to the admission webhook deployment object
func GetAdmissionReference(kubeClient kubernetes.Interface) (*metav1.OwnerReference, error) {

	// Fetch our namespace
	openebsNamespace, err := getOpenebsNamespace()
	if err != nil {
		return nil, err
	}

	// Fetch our admission server deployment object
	admdeployList, err := kubeClient.AppsV1().Deployments(openebsNamespace).
		List(context.TODO(), metav1.ListOptions{LabelSelector: webhookLabel})
	if err != nil {
		return nil, fmt.Errorf("failed to list admission deployment: %s", err.Error())
	}

	for _, admdeploy := range admdeployList.Items {
		if len(admdeploy.Name) != 0 {
			return metav1.NewControllerRef(admdeploy.GetObjectMeta(), schema.GroupVersionKind{
				Group:   appsv1.SchemeGroupVersion.Group,
				Version: appsv1.SchemeGroupVersion.Version,
				Kind:    "Deployment",
			}), nil

		}
	}
	return nil, fmt.Errorf("failed to create deployment ownerReference")
}

// preUpgrade checks for the required older webhook configs,older
// then 1.4.0 if exists delete them.
func (c *client) preUpgrade(openebsNamespace string) error {
	secretlist, err := c.kubeClient.CoreV1().Secrets(openebsNamespace).List(context.TODO(), metav1.ListOptions{LabelSelector: webhookLabel})
	if err != nil {
		return fmt.Errorf("failed to list old secret: %s", err.Error())
	}

	for _, scrt := range secretlist.Items {
		if scrt.Labels[types.OpenEBSVersionLabelKey] != version.Current() {
			newScrt := scrt
			for _, t := range transformSecret {
				t(&newScrt)
			}
			newScrt.Labels[types.OpenEBSVersionLabelKey] = version.Current()
			_, err = c.kubeClient.CoreV1().Secrets(openebsNamespace).Update(context.TODO(), &newScrt, metav1.UpdateOptions{})
			if err != nil {
				return fmt.Errorf("failed to update old secret %s: %s", scrt.Name, err.Error())
			}
		}
	}

	svcList, err := c.kubeClient.CoreV1().Services(openebsNamespace).List(context.TODO(), metav1.ListOptions{LabelSelector: webhooksvcLabel})
	if err != nil {
		return fmt.Errorf("failed to list old service: %s", err.Error())
	}

	for _, service := range svcList.Items {
		if service.Labels[types.OpenEBSVersionLabelKey] != version.Current() {
			newSvc := service
			for _, t := range transformSvc {
				t(&newSvc)
			}
			newSvc.Labels[types.OpenEBSVersionLabelKey] = version.Current()
			_, err = c.kubeClient.CoreV1().Services(openebsNamespace).Update(context.TODO(), &newSvc, metav1.UpdateOptions{})
			if err != nil {
				return fmt.Errorf("failed to update old service %s: %s", service.Name, err.Error())
			}
		}
	}
	webhookConfigList, err := c.kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().
		List(context.TODO(), metav1.ListOptions{LabelSelector: webhookLabel})
	if err != nil {
		return fmt.Errorf("failed to list older webhook config: %s", err.Error())
	}

	for _, config := range webhookConfigList.Items {
		if config.Labels[types.OpenEBSVersionLabelKey] != version.Current() {
			if IsCurrentLessThanNewVersion(config.Labels[string(types.OpenEBSVersionLabelKey)], "2.8.0") {
				err = c.kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Delete(context.TODO(), config.Name, metav1.DeleteOptions{})
				if err != nil {
					return fmt.Errorf("failed to delete older webhook config %s: %s", config.Name, err.Error())
				}
			} else {
				newConfig := config
				for _, t := range transformConfig {
					t(&newConfig)
				}
				newConfig.Labels[types.OpenEBSVersionLabelKey] = version.Current()
				_, err = c.kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().
					Update(context.TODO(), &newConfig, metav1.UpdateOptions{})
				if err != nil {
					return fmt.Errorf("failed to update older webhook config %s: %s", config.Name, err.Error())
				}
			}
		}
	}

	return nil
}

// failurePolicy returns the admission webhook configuration failurePolicy
// based on the given WebhookFailurePolicy ENV set on admission server
// deployments.
//
// Default failure Policy is `Fail` if not provided.
func failurePolicy() *admissionregistration.FailurePolicyType {
	var policyType *admissionregistration.FailurePolicyType
	policy, present := os.LookupEnv(WebhookFailurePolicy)
	if !present {
		policyType = &Fail
	}

	switch strings.ToLower(policy) {
	default:
		policyType = &Fail
	case "no", "false", "ignore":
		policyType = &Ignore
	}
	klog.Infof("Using webhook configuration failure policy as %q", *policyType)
	return policyType
}
