module github.com/openebs/cstor-operators

go 1.13

require (
	github.com/ahmetb/gen-crd-api-reference-docs v0.1.5
	github.com/davecgh/go-spew v1.1.1
	github.com/evanphx/json-patch v4.5.0+incompatible // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/imdario/mergo v0.3.8 // indirect
	github.com/onsi/ginkgo v1.12.0 // indirect
	github.com/onsi/gomega v1.9.0 // indirect
	github.com/openebs/api v1.10.0-RC1.0.20200608150240-08b494f77b77
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.4.0
	github.com/ugorji/go/codec v0.0.0-20181204163529-d75b2dcb6bc8
	go.uber.org/zap v1.13.0
	golang.org/x/net v0.0.0-20191004110552-13f9640d40b9
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d // indirect
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0 // indirect
	google.golang.org/grpc v1.23.1
	k8s.io/api v0.17.3
	k8s.io/apimachinery v0.17.3
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/klog v1.0.0
	k8s.io/kubernetes v1.17.3
	k8s.io/utils v0.0.0-20200124190032-861946025e34 // indirect
)

replace (
	k8s.io/api => k8s.io/api v0.17.3
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.17.3

	k8s.io/apimachinery => k8s.io/apimachinery v0.17.4-beta.0

	k8s.io/apiserver => k8s.io/apiserver v0.17.3

	k8s.io/cli-runtime => k8s.io/cli-runtime v0.17.3

	k8s.io/client-go => k8s.io/client-go v0.17.3

	k8s.io/cloud-provider => k8s.io/cloud-provider v0.17.3

	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.17.3

	k8s.io/code-generator => k8s.io/code-generator v0.17.4-beta.0

	k8s.io/component-base => k8s.io/component-base v0.17.3

	k8s.io/cri-api => k8s.io/cri-api v0.17.4-beta.0

	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.17.3

	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.17.3

	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.17.3

	k8s.io/kube-proxy => k8s.io/kube-proxy v0.17.3

	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.17.3

	k8s.io/kubectl => k8s.io/kubectl v0.17.3

	k8s.io/kubelet => k8s.io/kubelet v0.17.3

	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.17.3

	k8s.io/metrics => k8s.io/metrics v0.17.3

	k8s.io/node-api => k8s.io/node-api v0.17.3

	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.17.3

	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.17.3

	k8s.io/sample-controller => k8s.io/sample-controller v0.17.3
)
