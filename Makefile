# Copyright © 2020 The OpenEBS Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# The images can be pushed to any docker/image registeries
# like docker hub, quay. The registries are specified in
# the `build/push` script.
#
# The images of a project or company can then be grouped
# or hosted under a unique organization key like `openebs`
#
# Each component (container) will be pushed to a unique
# repository under an organization.
# Putting all this together, an unique uri for a given
# image comprises of:
#   <registry url>/<image org>/<image repo>:<image-tag>
#
# IMAGE_ORG can be used to customize the organization
# under which images should be pushed.
# By default the organization name is `openebs`.

ifeq (${IMAGE_ORG}, )
  IMAGE_ORG="openebs"
  export IMAGE_ORG
endif

# Specify the docker arg for repository url
ifeq (${DBUILD_REPO_URL}, )
  DBUILD_REPO_URL="https://github.com/openebs/cstor-operators"
  export DBUILD_REPO_URL
endif

# Specify the docker arg for website url
ifeq (${DBUILD_SITE_URL}, )
  DBUILD_SITE_URL="https://openebs.io"
  export DBUILD_SITE_URL
endif

## Specify the KUBECONFIG_PATH for running integration tests
ifeq (${KUBECONFIG_PATH},)
  KUBECONFIG_PATH="${HOME}/.kube/config"
  export KUBECONFIG_PATH
endif

# Specify the date of build
DBUILD_DATE=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')


# Determine the arch/os
ifeq (${XC_OS}, )
  XC_OS:=$(shell go env GOOS)
endif
export XC_OS

ifeq (${XC_ARCH}, )
  XC_ARCH:=$(shell go env GOARCH)
endif
export XC_ARCH

ARCH:=${XC_OS}_${XC_ARCH}
export ARCH

ifeq (${IMAGE_TAG}, )
  IMAGE_TAG = ci
  export IMAGE_TAG
endif

ifeq (${RELEASE_TAG}, )
  BASE_TAG = ci
  export BASE_TAG
else
  BASE_TAG = $(RELEASE_TAG:v%=%)
  export BASE_TAG
endif

export DBUILD_ARGS=--build-arg DBUILD_DATE=${DBUILD_DATE} --build-arg DBUILD_REPO_URL=${DBUILD_REPO_URL} --build-arg DBUILD_SITE_URL=${DBUILD_SITE_URL} --build-arg ARCH=${ARCH}

# Specify the name of cstor-base image
CSTOR_BASE_IMAGE= ${IMAGE_ORG}/cstor-base:${BASE_TAG}
export CSTOR_BASE_IMAGE

# Specify the name of the docker repo
CSPC_OPERATOR_REPO_NAME=cspc-operator
CVC_OPERATOR_REPO_NAME=cvc-operator
POOL_MANAGER_REPO_NAME=cstor-pool-manager
VOLUME_MANAGER_REPO_NAME=cstor-volume-manager
CSTOR_WEBHOOK_REPO_NAME=cstor-webhook

# Specify the directory location of main package after bin directory
# e.g. bin/{DIRECTORY_NAME_OF_APP}
CSPC_OPERATOR=cspc-operator
POOL_MANAGER=pool-manager
CVC_OPERATOR=cvc-operator
VOLUME_MANAGER=volume-manager
CSTOR_WEBHOOK=cstor-webhook
WEBHOOK_REPO=webhook

# list only the source code directories
PACKAGES = $(shell go list ./... | grep -v 'vendor\|pkg/client/generated\|tests')

# deps ensures fresh go.mod and go.sum.
.PHONY: deps
deps:
	@go mod tidy
	@go mod verify

.PHONY: test
test:
	go fmt ./...
	@echo "--> Running go test" ;
	@go test $(PACKAGES)

.PHONY: build
build:
	go build ./cmd/...

cvc-operator:
	@echo -n "--> cvc-operator <--"
	@echo "    "
	@PNAME=${CVC_OPERATOR} CTLNAME=${CVC_OPERATOR} sh -c "'$(PWD)/build/build.sh'"

.PHONY: cvc-operator-image
cvc-operator-image:
	@echo -n "--> cvc-operator image <--"
	@echo "${IMAGE_ORG}/${CVC_OPERATOR_REPO_NAME}:${IMAGE_TAG}"
	@echo "----------------------------"
	@PNAME=${CVC_OPERATOR} CTLNAME=${CVC_OPERATOR} sh -c "'$(PWD)/build/build.sh'"
	@cp bin/${CVC_OPERATOR}/${CVC_OPERATOR} build/cvc-operator/
	@cd build/${CVC_OPERATOR} && sudo docker build -t ${IMAGE_ORG}/${CVC_OPERATOR_REPO_NAME}:${IMAGE_TAG} ${DBUILD_ARGS} .
	@rm build/${CVC_OPERATOR}/${CVC_OPERATOR}

.PHONY: volume-manager-image
volume-manager-image:
	@echo -n "--> volume manager image <--"
	@echo "${IMAGE_ORG}/${VOLUME_MANAGER_REPO_NAME}:${IMAGE_TAG}"
	@echo "----------------------------"
	@PNAME=${VOLUME_MANAGER} CTLNAME=${VOLUME_MANAGER} sh -c "'$(PWD)/build/build.sh'"
	@cp bin/${VOLUME_MANAGER}/${VOLUME_MANAGER} build/volume-manager/
	@cd build/${VOLUME_MANAGER} && sudo docker build -t ${IMAGE_ORG}/${VOLUME_MANAGER_REPO_NAME}:${IMAGE_TAG} ${DBUILD_ARGS} .
	@rm build/${VOLUME_MANAGER}/${VOLUME_MANAGER}

.PHONY: cspc-operator-image
cspc-operator-image:
	@echo -n "--> cspc-operator image <--"
	@echo "${IMAGE_ORG}/${CSPC_OPERATOR_REPO_NAME}:${IMAGE_TAG}"
	@echo "----------------------------"
	@PNAME=${CSPC_OPERATOR} CTLNAME=${CSPC_OPERATOR} sh -c "'$(PWD)/build/build.sh'"
	@cp bin/${CSPC_OPERATOR}/${CSPC_OPERATOR} build/cspc-operator/
	@cd build/${CSPC_OPERATOR} && sudo docker build -t ${IMAGE_ORG}/${CSPC_OPERATOR_REPO_NAME}:${IMAGE_TAG} ${DBUILD_ARGS} .
	@rm build/${CSPC_OPERATOR}/${CSPC_OPERATOR}

.PHONY: pool-manager-image
pool-manager-image:
	@echo -n "--> pool manager image <--"
	@echo "${IMAGE_ORG}/${POOL_MANAGER_REPO_NAME}:${IMAGE_TAG}"
	@echo "----------------------------"
	@PNAME=${POOL_MANAGER} CTLNAME=${POOL_MANAGER} sh -c "'$(PWD)/build/build.sh'"
	@cp bin/${POOL_MANAGER}/${POOL_MANAGER} build/pool-manager/
	@cd build/${POOL_MANAGER} && sudo docker build -t ${IMAGE_ORG}/${POOL_MANAGER_REPO_NAME}:${IMAGE_TAG} --build-arg BASE_IMAGE=${CSTOR_BASE_IMAGE} ${DBUILD_ARGS} . --no-cache
	@rm build/${POOL_MANAGER}/${POOL_MANAGER}

.PHONY: cstor-webhook-image
cstor-webhook-image:
	@echo "----------------------------"
	@echo -n "--> cstor-webhook image "
	@echo "${IMAGE_ORG}/${CSTOR_WEBHOOK_REPO_NAME}:${IMAGE_TAG}"
	@echo "----------------------------"
	@PNAME=${CSTOR_WEBHOOK} CTLNAME=${WEBHOOK_REPO} sh -c "'$(PWD)/build/build.sh'"
	@cp bin/${CSTOR_WEBHOOK}/${WEBHOOK_REPO} build/cstor-webhook/
	@cd build/${CSTOR_WEBHOOK} && sudo docker build -t ${IMAGE_ORG}/${CSTOR_WEBHOOK_REPO_NAME}:${IMAGE_TAG} ${DBUILD_ARGS} .
	@rm build/${CSTOR_WEBHOOK}/${WEBHOOK_REPO}

.PHONY: all
all: cspc-operator-image pool-manager-image cstor-webhook-image \
           cvc-operator-image volume-manager-image

# Push images
.PHONY: deploy-images
deploy-images:
	@DIMAGE=${IMAGE_ORG}/cvc-operator-${XC_ARCH} ./build/push;
	@DIMAGE=${IMAGE_ORG}/cspc-operator-${XC_ARCH} ./build/push;
	@DIMAGE=${IMAGE_ORG}/cstor-volume-manager-${XC_ARCH} ./build/push;
	@DIMAGE=${IMAGE_ORG}/cstor-pool-manager-${XC_ARCH} ./build/push;
	@DIMAGE=${IMAGE_ORG}/cstor-webhook-${XC_ARCH} ./build/push;

.PHONY: gen-api-docs
gen-api-docs:
	@echo ">> generating cstor 'v1' apis docs"
	go run github.com/ahmetb/gen-crd-api-reference-docs -api-dir ../api/pkg/apis/cstor/v1 -config hack/api-docs/config.json -template-dir hack/api-docs/template -out-file docs/api-references/apis.md

manifests:
	@echo "-----------------------------------------------------"
	@echo "---   Generating cStor-operatory YAML     ----------"
	@echo "-----------------------------------------------------"
	./build/generate-manifest.sh

.PHONY: license-check
license-check:
	@echo "--> Checking license header..."
	@licRes=$$(for file in $$(find . -type f -regex '.*\.sh\|.*\.go\|.*Docker.*\|.*\Makefile*' ! -path './vendor/*' ) ; do \
               awk 'NR<=5' $$file | grep -Eq "(Copyright|generated|GENERATED)" || echo $$file; \
       done); \
       if [ -n "$${licRes}" ]; then \
               echo "license header checking failed:"; echo "$${licRes}"; \
               exit 1; \
       fi
	@echo "--> Done checking license."
	@echo
# If there are any external tools need to be used, they can be added by defining a EXTERNAL_TOOLS variable
# Bootstrap the build by downloading additional tools
.PHONY: bootstrap
bootstrap:
	@for tool in  $(EXTERNAL_TOOLS) ; do \
		echo "+ Installing $$tool" ; \
		cd && GO111MODULE=on go get $$tool; \
	done

.PHONY: clean
clean:
	@echo '--> Cleaning directory...'
	rm -rf ${GOPATH}/bin/${CSPC_OPERATOR}
	rm -rf ${GOPATH}/bin/${CVC_OPERATOR}
	rm -rf ${GOPATH}/bin/${POOL_MANAGER}
	rm -rf ${GOPATH}/bin/${VOLUME_MANAGER}
	rm -rf ${GOPATH}/bin/${CSTOR_WEBHOOK}
	@echo '--> Done cleaning.'
	@echo

include Makefile.buildx.mk

.PHONY: k8s-deploy
k8s-deploy:
	kubectl apply -f https://openebs.github.io/charts/openebs-operator.yaml
	kubectl apply -f https://openebs.github.io/charts/cstor-operator.yaml

.PHONY: k8s-deploy-devel
k8s-deploy-devel:
	kubectl apply -f deploy/yamls/rbac.yaml
	kubectl apply -f deploy/yamls/ndm-operator.yaml
	kubectl apply -f deploy/crds
	kubectl apply -f deploy/yamls/cspc-operator.yaml
	kubectl apply -f deploy/yamls/csi-operator.yaml

.PHONY: integration-test
integration-test:
	@echo "Running CStor Pool related tests"
	@echo "It is required to have minimum 15 Blockdevices to run integration test"
	go test ./tests/cspc/provisioning/... -v -timeout 60m -kubeconfig ${KUBECONFIG_PATH}
