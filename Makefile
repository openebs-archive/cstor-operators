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

ifeq (${TRAVIS_TAG}, )
  BASE_TAG = ci
  export BASE_TAG
else
  BASE_TAG = $(TRAVIS_TAG:v%=%)
  export BASE_TAG
endif

export DBUILD_ARGS=--build-arg DBUILD_DATE=${DBUILD_DATE} --build-arg DBUILD_REPO_URL=${DBUILD_REPO_URL} --build-arg DBUILD_SITE_URL=${DBUILD_SITE_URL} --build-arg ARCH=${ARCH}

# Specify the name of cstor-base image
CSTOR_BASE_IMAGE= ${IMAGE_ORG}/cstor-base:${BASE_TAG}
export CSTOR_BASE_IMAGE

ifeq (${CSTOR_BASE_IMAGE_ARM64}, )
  CSTOR_BASE_IMAGE_ARM64= ${IMAGE_ORG}/cstor-base-arm64:${BASE_TAG}
  export CSTOR_BASE_IMAGE_ARM64
endif

# Specify the name of base image for ARM64
ifeq (${BASE_DOCKER_IMAGE_ARM64}, )
  BASE_DOCKER_IMAGE_ARM64 = "arm64v8/ubuntu:18.04"
  export BASE_DOCKER_IMAGE_ARM64
endif

# Specify the name of the docker repo for amd64
CSPC_OPERATOR_REPO_NAME=cspc-operator-amd64
CVC_OPERATOR_REPO_NAME=cvc-operator-amd64
POOL_MANAGER_REPO_NAME=cstor-pool-manager-amd64
VOLUME_MANAGER_REPO_NAME=cstor-volume-manager-amd64
CSTOR_WEBHOOK_REPO_NAME=cstor-webhook-amd64

# Specify the directory location of main package after bin directory
# e.g. bin/{DIRECTORY_NAME_OF_APP}
CSPC_OPERATOR=cspc-operator
POOL_MANAGER=pool-manager
CVC_OPERATOR=cvc-operator
VOLUME_MANAGER=volume-manager
CSTOR_WEBHOOK=cstor-webhook
WEBHOOK_REPO=webhook
# Specify the name of the docker repo for arm64
CVC_OPERATOR_ARM64?=cvc-operator-arm64

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
	## To verify how much time it is consuming for test
	date
	go test $(PACKAGES)
	date

.PHONY: build
build:
	go build ./cmd/...

cvc-operator:
	@echo -n "--> cvc-operator <--"
	@echo "    "
	@PNAME=${CVC_OPERATOR} CTLNAME=${CVC_OPERATOR} sh -c "'$(PWD)/build/build.sh'"

.PHONY: cvc-operator-image.amd64
cvc-operator-image.amd64:
	@echo -n "--> cvc-operator image <--"
	@echo "${IMAGE_ORG}/${CVC_OPERATOR_REPO_NAME}:${IMAGE_TAG}"
	@echo "----------------------------"
	@PNAME=${CVC_OPERATOR} CTLNAME=${CVC_OPERATOR} sh -c "'$(PWD)/build/build.sh'"
	@cp bin/${CVC_OPERATOR}/${CVC_OPERATOR} build/cvc-operator/
	@cd build/${CVC_OPERATOR} && sudo docker build -t ${IMAGE_ORG}/${CVC_OPERATOR_REPO_NAME}:${IMAGE_TAG} ${DBUILD_ARGS} .
	@rm build/${CVC_OPERATOR}/${CVC_OPERATOR}

.PHONY: cvc-operator-image.arm64
cvc-operator-image.arm64:
	@echo "----------------------------"
	@echo -n "--> arm64 based cvc-operator image "
	@echo "${IMAGE_ORG}/${CVC_OPERATOR_REPO_NAME}:${IMAGE_TAG}"
	@echo "----------------------------"
	@PNAME=${CVC_OPERATOR} CTLNAME=${CVC_OPERATOR} sh -c "'$(PWD)/build/build.sh'"
	@cp bin/${CVC_OPERATOR}/${CVC_OPERATOR} build/cvc-operator/
	@cd build/${CVC_OPERATOR} && sudo docker build -t ${IMAGE_ORG}/${CVC_OPERATOR_ARM64}:${IMAGE_TAG} -f Dockerfile.arm64 ${DBUILD_ARGS} .
	@rm build/${CVC_OPERATOR}/${CVC_OPERATOR}

.PHONY: volume-manager-image.amd64
volume-manager-image.amd64:
	@echo -n "--> volume manager image <--"
	@echo "${IMAGE_ORG}/${VOLUME_MANAGER_REPO_NAME}:${IMAGE_TAG}"
	@echo "----------------------------"
	@PNAME=${VOLUME_MANAGER} CTLNAME=${VOLUME_MANAGER} sh -c "'$(PWD)/build/build.sh'"
	@cp bin/${VOLUME_MANAGER}/${VOLUME_MANAGER} build/volume-manager/
	@cd build/${VOLUME_MANAGER} && sudo docker build -t ${IMAGE_ORG}/${VOLUME_MANAGER_REPO_NAME}:${IMAGE_TAG} ${DBUILD_ARGS} .
	@rm build/${VOLUME_MANAGER}/${VOLUME_MANAGER}

.PHONY: cspc-operator-image.amd64
cspc-operator-image.amd64:
	@echo -n "--> cspc-operator image <--"
	@echo "${IMAGE_ORG}/${CSPC_OPERATOR_REPO_NAME}:${IMAGE_TAG}"
	@echo "----------------------------"
	@PNAME=${CSPC_OPERATOR} CTLNAME=${CSPC_OPERATOR} sh -c "'$(PWD)/build/build.sh'"
	@cp bin/${CSPC_OPERATOR}/${CSPC_OPERATOR} build/cspc-operator/
	@cd build/${CSPC_OPERATOR} && sudo docker build -t ${IMAGE_ORG}/${CSPC_OPERATOR_REPO_NAME}:${IMAGE_TAG} ${DBUILD_ARGS} .
	@rm build/${CSPC_OPERATOR}/${CSPC_OPERATOR}

.PHONY: pool-manager-image.amd64
pool-manager-image.amd64:
	@echo -n "--> pool manager image <--"
	@echo "${IMAGE_ORG}/${POOL_MANAGER_REPO_NAME}:${IMAGE_TAG}"
	@echo "----------------------------"
	@PNAME=${POOL_MANAGER} CTLNAME=${POOL_MANAGER} sh -c "'$(PWD)/build/build.sh'"
	@cp bin/${POOL_MANAGER}/${POOL_MANAGER} build/pool-manager/
	@cd build/${POOL_MANAGER} && sudo docker build -t ${IMAGE_ORG}/${POOL_MANAGER_REPO_NAME}:${IMAGE_TAG} --build-arg BASE_IMAGE=${CSTOR_BASE_IMAGE} ${DBUILD_ARGS} . --no-cache
	@rm build/${POOL_MANAGER}/${POOL_MANAGER}

.PHONY: cstor-webhook-image.amd64
cstor-webhook-image.amd64:
	@echo "----------------------------"
	@echo -n "--> cstor-webhook image "
	@echo "${IMAGE_ORG}/${CSTOR_WEBHOOK_REPO_NAME}:${IMAGE_TAG}"
	@echo "----------------------------"
	@PNAME=${CSTOR_WEBHOOK} CTLNAME=${WEBHOOK_REPO} sh -c "'$(PWD)/build/build.sh'"
	@cp bin/${CSTOR_WEBHOOK}/${WEBHOOK_REPO} build/cstor-webhook/
	@cd build/${CSTOR_WEBHOOK} && sudo docker build -t ${IMAGE_ORG}/${CSTOR_WEBHOOK_REPO_NAME}:${IMAGE_TAG} ${DBUILD_ARGS} .
	@rm build/${CSTOR_WEBHOOK}/${WEBHOOK_REPO}

.PHONY: all.amd64
all.amd64: cspc-operator-image.amd64 pool-manager-image.amd64 cstor-webhook-image.amd64 \
           cvc-operator-image.amd64 volume-manager-image.amd64

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
