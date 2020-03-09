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
	"fmt"
	"os"
	"strings"

	"github.com/openebs/api/pkg/apis/types"
	"github.com/openebs/api/pkg/util"
	"github.com/openebs/maya/pkg/version"
	"github.com/pkg/errors"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

const (
	validatorServiceName = "admission-server-svc"
	validatorWebhook     = "openebs-validation-webhook-cfg"
	validatorSecret      = "admission-server-secret"
	webhookHandlerName   = "admission-webhook.openebs.io"
	validationPath       = "/validate"
	validationPort       = 8443
	webhookLabel         = "openebs.io/component-name" + "=" + "admission-webhook"
	webhooksvcLabel      = "openebs.io/component-name" + "=" + "admission-webhook-svc"
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
type transformConfigFunc func(*admissionregistrationv1.ValidatingWebhookConfiguration)

var (
	// TimeoutSeconds specifies the timeout for this webhook. After the timeout passes,
	// the webhook call will be ignored or the API call will fail based on the
	// failure policy.
	// The timeout value must be between 1 and 30 seconds.
	five = int32(5)
	// Ignore means that an error calling the webhook is ignored.
	Ignore = admissionregistrationv1.Ignore
	// transformation function lists to upgrade webhook resources
	transformSecret = []transformSecretFunc{}
	transformSvc    = []transformSvcFunc{}
	transformConfig = []transformConfigFunc{}
)

// createWebhookService creates our webhook Service resource if it does not
// exist.
func (c *client) createWebhookService(
	ownerReference metav1.OwnerReference,
	serviceName string,
	namespace string,
) error {

	_, err := c.kubeClient.CoreV1().Services(namespace).
		Get(serviceName, metav1.GetOptions{})

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
	serviceLabels := map[string]string{"app": "admission-webhook"}
	svcObj := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      serviceName,
			Labels: map[string]string{
				"app":                                "admission-webhook",
				"openebs.io/component-name":          "admission-webhook-svc",
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
		Create(svcObj)
	return err
}

// createAdmissionService creates our ValidatingWebhookConfiguration resource
// if it does not exist.
func (c *client) createAdmissionService(
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
			"failed to get webhook validator {%v}",
			validatorWebhook,
		)
	}

	sideEffectClass := admissionregistrationv1.SideEffectClassNone

	webhookHandler := admissionregistrationv1.ValidatingWebhook{
		Name: webhookHandlerName,
		Rules: []admissionregistrationv1.RuleWithOperations{{
			Operations: []admissionregistrationv1.OperationType{
				admissionregistrationv1.Create,
				admissionregistrationv1.Delete,
			},
			Rule: admissionregistrationv1.Rule{
				APIGroups:   []string{"*"},
				APIVersions: []string{"*"},
				Resources:   []string{"persistentvolumeclaims"},
			},
		},
			{
				Operations: []admissionregistrationv1.OperationType{
					admissionregistrationv1.Create,
					admissionregistrationv1.Update,
					admissionregistrationv1.Delete,
				},
				Rule: admissionregistrationv1.Rule{
					APIGroups:   []string{"*"},
					APIVersions: []string{"*"},
					Resources:   []string{"cstorpoolclusters"},
				},
			}},
		ClientConfig: admissionregistrationv1.WebhookClientConfig{
			Service: &admissionregistrationv1.ServiceReference{
				Namespace: namespace,
				Name:      serviceName,
				Path:      StrPtr(validationPath),
			},
			CABundle: signingCert,
		},
		SideEffects:             &sideEffectClass,
		AdmissionReviewVersions: []string{"v1", "v1beta1"},
		TimeoutSeconds:          &five,
		FailurePolicy:           &Ignore,
	}

	validator := &admissionregistrationv1.ValidatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{
			Kind:       "validatingWebhookConfiguration",
			APIVersion: "admissionregistration.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: validatorWebhook,
			Labels: map[string]string{
				"app":                                "admission-webhook",
				"openebs.io/component-name":          "admission-webhook",
				string(types.OpenEBSVersionLabelKey): version.GetVersion(),
			},
			OwnerReferences: []metav1.OwnerReference{ownerReference},
		},
		Webhooks: []admissionregistrationv1.ValidatingWebhook{webhookHandler},
	}

	_, err = c.kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().
		Create(validator)

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
				"app":                                "admission-webhook",
				"openebs.io/component-name":          "admission-webhook",
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

	return c.kubeClient.CoreV1().Secrets(namespace).Create(secretObj)
}

// GetValidatorWebhook fetches the webhook validator resource in
// Openebs namespace.
func GetValidatorWebhook(
	validator string, kubeClient kubernetes.Interface,
) (*admissionregistrationv1.ValidatingWebhookConfiguration, error) {

	return kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(validator, metav1.GetOptions{})
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

	validatorErr := c.createAdmissionService(
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

	return kubeClient.CoreV1().Secrets(namespace).Get(secretName, metav1.GetOptions{})
}

// getOpenebsNamespace gets the namespace OPENEBS_NAMESPACE env value which is
// set by the downward API where admission server has been deployed
func getOpenebsNamespace() (string, error) {

	ns := util.LookupOrFalse(util.OpenEBSNamespace)
	if ns == "false" {
		return "", fmt.Errorf("%s must be set", util.OpenEBSNamespace)
	}
	return ns, nil
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
		List(metav1.ListOptions{LabelSelector: webhookLabel})
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
	secretlist, err := c.kubeClient.CoreV1().Secrets(openebsNamespace).List(metav1.ListOptions{LabelSelector: webhookLabel})
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
			_, err = c.kubeClient.CoreV1().Secrets(openebsNamespace).Update(&newScrt)
			if err != nil {
				return fmt.Errorf("failed to update old secret %s: %s", scrt.Name, err.Error())
			}
		}
	}

	svcList, err := c.kubeClient.CoreV1().Services(openebsNamespace).List(metav1.ListOptions{LabelSelector: webhooksvcLabel})
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
			_, err = c.kubeClient.CoreV1().Services(openebsNamespace).Update(&newSvc)
			if err != nil {
				return fmt.Errorf("failed to update old service %s: %s", service.Name, err.Error())
			}
		}
	}
	webhookConfigList, err := c.kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().
		List(metav1.ListOptions{LabelSelector: webhookLabel})
	if err != nil {
		return fmt.Errorf("failed to list older webhook config: %s", err.Error())
	}

	for _, config := range webhookConfigList.Items {
		if config.Labels[types.OpenEBSVersionLabelKey] != version.Current() {
			newConfig := config
			for _, t := range transformConfig {
				t(&newConfig)
			}
			newConfig.Labels[types.OpenEBSVersionLabelKey] = version.Current()
			_, err = c.kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().
				Update(&newConfig)
			if err != nil {
				return fmt.Errorf("failed to update older webhook config %s: %s", config.Name, err.Error())
			}
		}
	}

	return nil
}
