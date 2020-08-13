# Copyright 2018-2020 The OpenEBS Authors. All rights reserved.
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

# ==============================================================================
# Build Options

export DBUILD_ARGS=--build-arg DBUILD_DATE=${DBUILD_DATE} --build-arg DBUILD_REPO_URL=${DBUILD_REPO_URL} --build-arg DBUILD_SITE_URL=${DBUILD_SITE_URL}

ifeq (${TAG}, )
  export TAG=ci
endif

# default list of platforms for which multiarch image is built
ifeq (${PLATFORMS}, )
	export PLATFORMS="linux/amd64,linux/arm64,linux/arm/v7,linux/ppc64le"
endif

# if IMG_RESULT is unspecified, by default the image will be pushed to registry
ifeq (${IMG_RESULT}, load)
	export PUSH_ARG="--load"
    # if load is specified, image will be built only for the build machine architecture.
    export PLATFORMS="local"
else ifeq (${IMG_RESULT}, cache)
	# if cache is specified, image will only be available in the build cache, it won't be pushed or loaded
	# therefore no PUSH_ARG will be specified
else
	export PUSH_ARG="--push"
endif

# Name of the multiarch image for cspc-operator
DOCKERX_IMAGE_CSPC_OPERATOR:=${IMAGE_ORG}/cspc-operator:${TAG}

# Name of the multiarch image for cvc-operator
DOCKERX_IMAGE_CVC_OPERATOR:=${IMAGE_ORG}/cvc-operator:${TAG}

# Name of the multiarch image for pool-manager
DOCKERX_IMAGE_POOL_MANAGER:=${IMAGE_ORG}/pool-manager:${TAG}

# Name of the multiarch image for volume-manager
DOCKERX_IMAGE_VOLUME_MANAGER:=${IMAGE_ORG}/volume-manager:${TAG}

# Name of the multiarch image for cstor-webhook
DOCKERX_IMAGE_CSTOR_WEBHOOK:=${IMAGE_ORG}/cstor-webhook:${TAG}


# Build cstor-operator docker images with buildx
# Experimental docker feature to build cross platform multi-architecture docker images
# https://docs.docker.com/buildx/working-with-buildx/
.PHONY: buildx.cspc-operator
buildx.cspc-operator: bootstrap clean 
	@echo '--> Building cspc-operator binary...'
	@pwd
	@PNAME=${CSPC_OPERATOR} CTLNAME=${CSPC_OPERATOR} BUILDX=true sh -c "'$(PWD)/build/build.sh'"
	@echo '--> Built binary.'
	@echo

.PHONY: docker.buildx.cspc-operator
docker.buildx.cspc-operator:
	export DOCKER_CLI_EXPERIMENTAL=enabled
	@if ! docker buildx ls | grep -q container-builder; then\
		docker buildx create --platform ${PLATFORMS} --name container-builder --use;\
	fi
	@docker buildx build --platform ${PLATFORMS} \
		-t "$(DOCKERX_IMAGE_CSPC_OPERATOR)" ${DBUILD_ARGS} -f $(PWD)/build/$(CSPC_OPERATOR)/cspc-operator.Dockerfile \
		. ${PUSH_ARG}
	@echo "--> Build docker image: $(DOCKERX_IMAGE_CSPC_OPERATOR)"
	@echo

.PHONY: buildx.cvc-operator
buildx.cvc-operator: bootstrap clean 
	@echo '--> Building cvc-operator binary...'
	@pwd
	@PNAME=${CVC_OPERATOR} CTLNAME=${CVC_OPERATOR} BUILDX=true sh -c "'$(PWD)/build/build.sh'"
	@echo '--> Built binary.'
	@echo

.PHONY: docker.buildx.cvc-operator
docker.buildx.cvc-operator:
	export DOCKER_CLI_EXPERIMENTAL=enabled
	@if ! docker buildx ls | grep -q container-builder; then\
		docker buildx create --platform ${PLATFORMS} --name container-builder --use;\
	fi
	@docker buildx build --platform ${PLATFORMS} \
		-t "$(DOCKERX_IMAGE_CVC_OPERATOR)" ${DBUILD_ARGS} -f $(PWD)/build/$(CVC_OPERATOR)/cvc-operator.Dockerfile \
		. ${PUSH_ARG}
	@echo "--> Build docker image: $(DOCKERX_IMAGE_CVC_OPERATOR)"
	@echo

.PHONY: buildx.pool-manager
buildx.pool-manager: bootstrap clean 
	@echo '--> Building pool-manager binary...'
	@pwd
	@PNAME=${POOL_MANAGER} CTLNAME=${POOL_MANAGER} BUILDX=true sh -c "'$(PWD)/build/build.sh'"
	@echo '--> Built binary.'
	@echo

.PHONY: docker.buildx.pool-manager
docker.buildx.pool-manager:
	export DOCKER_CLI_EXPERIMENTAL=enabled
	@if ! docker buildx ls | grep -q container-builder; then\
		docker buildx create --platform ${PLATFORMS} --name container-builder --use;\
	fi
	@docker buildx build --platform ${PLATFORMS} \
		-t "$(DOCKERX_IMAGE_POOL_MANAGER)" --build-arg BASE_IMAGE=${CSTOR_BASE_IMAGE} ${DBUILD_ARGS} -f $(PWD)/build/$(POOL_MANAGER)/pool-manager.Dockerfile \
		. ${PUSH_ARG}
	@echo "--> Build docker image: $(DOCKERX_IMAGE_POOL_MANAGER)"
	@echo

.PHONY: buildx.volume-manager
buildx.volume-manager: bootstrap clean 
	@echo '--> Building volume-manager binary...'
	@pwd
	@PNAME=${VOLUME_MANAGER} CTLNAME=${VOLUME_MANAGER} BUILDX=true sh -c "'$(PWD)/build/build.sh'"
	@echo '--> Built binary.'
	@echo

.PHONY: docker.buildx.volume-manager
docker.buildx.volume-manager:
	export DOCKER_CLI_EXPERIMENTAL=enabled
	@if ! docker buildx ls | grep -q container-builder; then\
		docker buildx create --platform ${PLATFORMS} --name container-builder --use;\
	fi
	@docker buildx build --platform ${PLATFORMS} \
		-t "$(DOCKERX_IMAGE_VOLUME_MANAGER)" ${DBUILD_ARGS} -f $(PWD)/build/$(VOLUME_MANAGER)/volume-manager.Dockerfile \
		. ${PUSH_ARG}
	@echo "--> Build docker image: $(DOCKERX_IMAGE_VOLUME_MANAGER)"
	@echo

.PHONY: buildx.cstor-webhook
buildx.cstor-webhook: bootstrap clean 
	@echo '--> Building cstor-webhook binary...'
	@pwd
	@PNAME=${CSTOR_WEBHOOK} CTLNAME=${WEBHOOK_REPO} BUILDX=true sh -c "'$(PWD)/build/build.sh'"
	@echo '--> Built binary.'
	@echo

.PHONY: docker.buildx.cstor-webhook
docker.buildx.cstor-webhook:
	export DOCKER_CLI_EXPERIMENTAL=enabled
	@if ! docker buildx ls | grep -q container-builder; then\
		docker buildx create --platform ${PLATFORMS} --name container-builder --use;\
	fi
	@docker buildx build --platform ${PLATFORMS} \
		-t "$(DOCKERX_IMAGE_CSTOR_WEBHOOK)" ${DBUILD_ARGS} -f $(PWD)/build/$(CSTOR_WEBHOOK)/cstor-webhook.Dockerfile \
		. ${PUSH_ARG}
	@echo "--> Build docker image: $(DOCKERX_IMAGE_CSTOR_WEBHOOK)"
	@echo

.PHONY: buildx.push.cspc-operator
buildx.push.cspc-operator:
	BUILDX=true DIMAGE=${IMAGE_ORG}/cspc-operator ./build/push

.PHONY: buildx.push.cvc-operator
buildx.push.cvc-operator:
	BUILDX=true DIMAGE=${IMAGE_ORG}/cvc-operator ./build/push

.PHONY: buildx.push.pool-manager
buildx.push.pool-manager:
	BUILDX=true DIMAGE=${IMAGE_ORG}/pool-manager ./build/push

.PHONY: buildx.push.volume-manager
buildx.push.volume-manager:
	BUILDX=true DIMAGE=${IMAGE_ORG}/volume-manager ./build/push

.PHONY: buildx.push.cstor-webhook
buildx.push.cstor-webhook:
	BUILDX=true DIMAGE=${IMAGE_ORG}/cstor-webhook ./build/push