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


HUB_USER?=openebs

ifeq (${IMAGE_TAG}, )
  IMAGE_TAG = ci
  export IMAGE_TAG
endif

ifeq (${TRAVIS_TAG}, )
  BASE_TAG = ci
  export BASE_TAG
else
  BASE_TAG = ${TRAVIS_TAG}
  export BASE_TAG
endif

# Specify the name of cstor-base image
CSTOR_BASE_IMAGE= openebs/cstor-base:${BASE_TAG}
export CSTOR_BASE_IMAGE


# Specify the name of the docker repo for amd64
CVC_OPERATOR?=cvc-operator
CSPC_OPERATOR=cspc-operator
POOL_MANAGER=pool-manager
VOLUME_MANAGER=volume-manager
# Specify the name of the docker repo for arm64
CVC_OPERATOR_ARM64?=cvc-operator-arm64


# deps ensures fresh go.mod and go.sum.
.PHONY: deps
deps:
	@go mod tidy
	@go mod verify

.PHONY: test
test:
	go test ./...

cvc-operator:
	@echo -n "--> cvc-operator <--"
	@echo "    "
	@PNAME=${CVC_OPERATOR} CTLNAME=${CVC_OPERATOR} sh -c "'$(PWD)/build/build.sh'"

.PHONY: cvc-operator-image
cvc-operator-image:
	@echo -n "--> cvc-operator image <--"
	@echo "${HUB_USER}/${CVC_OPERATOR_REPO_NAME}:${IMAGE_TAG}"
	@echo "----------------------------"
	@PNAME=${CVC_OPERATOR} CTLNAME=${CVC_OPERATOR} sh -c "'$(PWD)/build/build.sh'"
	@cp bin/${CVC_OPERATOR}/${CVC_OPERATOR} build/cvc-operator/
	@cd build/${CVC_OPERATOR} && sudo docker build -t ${HUB_USER}/${CVC_OPERATOR}:${IMAGE_TAG} --build-arg BUILD_DATE=${BUILD_DATE} .
	@rm build/${CVC_OPERATOR}/${CVC_OPERATOR}

.PHONY: cvc-operator-image.arm64
cvc-operator-image.arm64:
	@echo "----------------------------"
	@echo -n "--> arm64 based cvc-operator image "
	@echo "${HUB_USER}/${CVC_OPERATOR_REPO_NAME}:${IMAGE_TAG}"
	@echo "----------------------------"
	@PNAME=${CVC_OPERATOR} CTLNAME=${CVC_OPERATOR} sh -c "'$(PWD)/build/build.sh'"
	@cp bin/${CVC_OPERATOR}/${CVC_OPERATOR} build/cvc-operator/
	@cd build/${CVC_OPERATOR} && sudo docker build -t ${HUB_USER}/${CVC_OPERATOR_ARM64}:${IMAGE_TAG} -f Dockerfile.arm64 --build-arg BUILD_DATE=${BUILD_DATE} .
	@rm build/${CVC_OPERATOR}/${CVC_OPERATOR}

.PHONY: volume-manager-image
volume-manager-image:
	@echo -n "--> volume manager image <--"
	@echo "${HUB_USER}/${VOLUME_MANAGER}:${IMAGE_TAG}"
	@echo "----------------------------"
	@PNAME=${VOLUME_MANAGER} CTLNAME=${VOLUME_MANAGER} sh -c "'$(PWD)/build/build.sh'"
	@cp bin/${VOLUME_MANAGER}/${VOLUME_MANAGER} build/volume-manager/
	@cd build/${VOLUME_MANAGER} && sudo docker build -t ${HUB_USER}/${VOLUME_MANAGER}:${IMAGE_TAG} --build-arg BUILD_DATE=${BUILD_DATE} .
	@rm build/${VOLUME_MANAGER}/${VOLUME_MANAGER}

.PHONY: cspc-operator-image
cspc-operator-image:
	@echo -n "--> cspc-operator image <--"
	@echo "${HUB_USER}/${CSPC_OPERATOR}:${IMAGE_TAG}"
	@echo "----------------------------"
	@PNAME=${CSPC_OPERATOR} CTLNAME=${CSPC_OPERATOR} sh -c "'$(PWD)/build/build.sh'"
	@cp bin/${CSPC_OPERATOR}/${CSPC_OPERATOR} build/cspc-operator/
	@cd build/${CSPC_OPERATOR} && sudo docker build -t ${HUB_USER}/${CSPC_OPERATOR}:${IMAGE_TAG} --build-arg BUILD_DATE=${BUILD_DATE} .
	@rm build/${CSPC_OPERATOR}/${CSPC_OPERATOR}

.PHONY: pool-manager-image
pool-manager-image:
	@echo -n "--> pool manager image <--"
	@echo "${HUB_USER}/${POOL_MANAGER}:${IMAGE_TAG}"
	@echo "----------------------------"
	@PNAME=${POOL_MANAGER} CTLNAME=${POOL_MANAGER} sh -c "'$(PWD)/build/build.sh'"
	@cp bin/${POOL_MANAGER}/${POOL_MANAGER} build/pool-manager/
	@cd build/${POOL_MANAGER} && sudo docker build -t ${HUB_USER}/${POOL_MANAGER}:${IMAGE_TAG} --build-arg BUILD_DATE=${BUILD_DATE} .
	@rm build/${POOL_MANAGER}/${POOL_MANAGER}
