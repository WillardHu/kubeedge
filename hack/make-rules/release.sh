#!/usr/bin/env bash

###
#Copyright 2021 The KubeEdge Authors.
#
#Licensed under the Apache License, Version 2.0 (the "License");
#you may not use this file except in compliance with the License.
#You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
#Unless required by applicable law or agreed to in writing, software
#distributed under the License is distributed on an "AS IS" BASIS,
#WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#See the License for the specific language governing permissions and
#limitations under the License.
###

set -o errexit
set -o nounset
set -o pipefail

KUBEEDGE_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"

source "${KUBEEDGE_ROOT}/hack/lib/init.sh"

ALL_RELEASE_TARGETS=(
  "kubeedge"
  "edgesite"
  "keadm"
)

function release() {
  local -a targets=()
  local VERSION=""
  local ARCH="amd64"
  local OS="linux"
  local arm_version=""

  for arg in "$@"; do
    if [[ "${arg}" == GOARM7 ]]; then
      ARCH="arm"
      arm_version="GOARM7"
    elif [[ "${arg}" == GOARM8 ]]; then
      arm_version="GOARM8"
      ARCH="arm64"
    elif [[ "${arg}" == "linux" ]]; then
      OS="linux"
    elif [[ "${arg}" == windows ]]; then
      OS="windows"
    else
      targets+=("${arg}")
    fi
  done

  if [[ ${#targets[@]} -eq 0 ]]; then
    targets=("${ALL_RELEASE_TARGETS[@]}")
  fi

  kubeedge::version::get_version_info
  VERSION=${GIT_VERSION}

  for bin in ${targets[@]}; do
    case "${bin}" in
      "edgesite")
        if [ "${ARCH}" == "amd64" ]; then
          hack/make-rules/build.sh edgesite-server edgesite-agent
        else
          hack/make-rules/crossbuild.sh edgesite-server edgesite-agent ${arm_version}
        fi

        build_edgesite_release $VERSION $ARCH
        ;;
      "keadm")
        if [ "${ARCH}" == "amd64" ] && [ "${OS}" == "linux" ]; then
          hack/make-rules/build.sh keadm
        else
          hack/make-rules/crossbuild.sh keadm ${arm_version} GOARCH${ARCH} GOOS${OS}
        fi

        build_keadm_release $VERSION $ARCH $OS
        ;;
      "kubeedge")
        if [ "${OS}" == "linux" ]; then
          if [ "${ARCH}" == "amd64" ]; then
            hack/make-rules/build.sh cloudcore admission edgecore csidriver iptablesmanager controllermanager
          else
            hack/make-rules/crossbuild.sh cloudcore admission edgecore csidriver iptablesmanager controllermanager ${arm_version} GOOS${OS} GOARCH${ARCH}
          fi
        fi

        if [ "${OS}" == "windows" ]; then
          hack/make-rules/crossbuild.sh edgecore GOOS${OS} GOARCH${ARCH}
        fi

        build_kubeedge_release $VERSION $ARCH $OS
        ;;
      *)
        echo "not supported release:" $bin "only supported:" ${ALL_RELEASE_TARGETS[@]}
        exit 1
    esac
  done
}

function build_kubeedge_release() {
  local VERSION=""
  local ARCH="amd64"
  local OS="linux"

  for arg in "$@"; do
    if [[ "${arg}" == v* ]]; then
      VERSION="${arg}"
    elif [[ "${arg}" == arm* ]]; then
      ARCH="${arg}"
    elif [[ "${arg}" == "windows" ]]; then
      OS="windows"
    fi
  done

  echo "building kubeedge release:" ${VERSION} "ARCH:"${ARCH} "OS:"${OS}

  if [ "${OS}" == "linux" ]; then
    mkdir -p _output/release/${VERSION}/kubeedge-${VERSION}-${OS}-${ARCH}/cloud
    mkdir -p _output/release/${VERSION}/kubeedge-${VERSION}-${OS}-${ARCH}/cloud/admission
    mkdir -p _output/release/${VERSION}/kubeedge-${VERSION}-${OS}-${ARCH}/cloud/cloudcore
    mkdir -p _output/release/${VERSION}/kubeedge-${VERSION}-${OS}-${ARCH}/cloud/csidriver
    mkdir -p _output/release/${VERSION}/kubeedge-${VERSION}-${OS}-${ARCH}/cloud/iptablesmanager
    mkdir -p _output/release/${VERSION}/kubeedge-${VERSION}-${OS}-${ARCH}/cloud/controllermanager
    mkdir -p _output/release/${VERSION}/kubeedge-${VERSION}-${OS}-${ARCH}/edge

    cp _output/local/bin/admission _output/release/${VERSION}/kubeedge-${VERSION}-${OS}-${ARCH}/cloud/admission
    cp _output/local/bin/cloudcore _output/release/${VERSION}/kubeedge-${VERSION}-${OS}-${ARCH}/cloud/cloudcore
    cp _output/local/bin/csidriver _output/release/${VERSION}/kubeedge-${VERSION}-${OS}-${ARCH}/cloud/csidriver
    cp _output/local/bin/iptablesmanager _output/release/${VERSION}/kubeedge-${VERSION}-${OS}-${ARCH}/cloud/iptablesmanager
    cp _output/local/bin/controllermanager _output/release/${VERSION}/kubeedge-${VERSION}-${OS}-${ARCH}/cloud/controllermanager
    cp _output/local/bin/edgecore _output/release/${VERSION}/kubeedge-${VERSION}-${OS}-${ARCH}/edge

  elif [ "${OS}" == "windows" ]; then
    mkdir -p _output/release/${VERSION}/kubeedge-${VERSION}-${OS}-${ARCH}/edge
    cp _output/local/bin/edgecore _output/release/${VERSION}/kubeedge-${VERSION}-${OS}-${ARCH}/edge/edgecore.exe
  fi

  echo ${VERSION} > _output/release/${VERSION}/kubeedge-${VERSION}-${OS}-${ARCH}/version
  cd _output/release/${VERSION}
  tar -czvf ${KUBEEDGE_ROOT}/_output/release/${VERSION}/kubeedge-${VERSION}-${OS}-${ARCH}.tar.gz kubeedge-${VERSION}-${OS}-${ARCH}/

  cd $KUBEEDGE_ROOT
  rm -r _output/release/${VERSION}/kubeedge-${VERSION}-${OS}-${ARCH}/

  #calculate sha512sum
  #the below command got like this:
  # d6db3c28b1991de781bf19a82fc5b24a1dbf9634e8edfa10e2ad8636baaf37ab3141ea8814db1f1c91119fccc9b7ff44d8ab9f3def536fd5262418035f527e86  kubeedge-v1.9.0-linux-amd64.tar.gz
  sum=$(sha512sum _output/release/${VERSION}/kubeedge-${VERSION}-${OS}-${ARCH}.tar.gz)
  sumArray=($sum)
  echo ${sumArray[0]} > _output/release/${VERSION}/checksum_kubeedge-${VERSION}-${OS}-${ARCH}.tar.gz.txt
}

function build_keadm_release() {
  local VERSION=""
  local ARCH="amd64"
  local OS="linux"

  for arg in "$@"; do
    if [[ "${arg}" == v* ]]; then
      VERSION="${arg}"
    elif [[ "${arg}" == arm* ]]; then
      ARCH="${arg}"
    elif [[ "${arg}" == "windows" ]]; then
      OS="windows"
    fi
  done

  echo "building keadm release:" ${VERSION} "ARCH:"${ARCH} "OS:"${OS}

  mkdir -p _output/release/${VERSION}/keadm-${VERSION}-${OS}-${ARCH}/keadm

  echo ${VERSION} > _output/release/${VERSION}/keadm-${VERSION}-${OS}-${ARCH}/version
  if [ "${OS}" == "linux" ]; then
    cp _output/local/bin/keadm _output/release/${VERSION}/keadm-${VERSION}-${OS}-${ARCH}/keadm
  elif [ "${OS}" == "windows" ]; then
    cp _output/local/bin/keadm _output/release/${VERSION}/keadm-${VERSION}-${OS}-${ARCH}/keadm/keadm.exe
  fi

  cd _output/release/${VERSION}
  tar -czvf ${KUBEEDGE_ROOT}/_output/release/${VERSION}/keadm-${VERSION}-${OS}-${ARCH}.tar.gz keadm-${VERSION}-${OS}-${ARCH}/

  cd $KUBEEDGE_ROOT
  rm -r _output/release/${VERSION}/keadm-${VERSION}-${OS}-${ARCH}

  #calculate sha512sum
  sum=$(sha512sum _output/release/${VERSION}/keadm-${VERSION}-${OS}-${ARCH}.tar.gz)
  sumArray=($sum)
  echo ${sumArray[0]} > _output/release/${VERSION}/checksum_keadm-${VERSION}-${OS}-${ARCH}.tar.gz.txt
}

function build_edgesite_release() {
  local VERSION=""
  local ARCH="amd64"

  for arg in "$@"; do
    if [[ "${arg}" == v* ]]; then
      VERSION="${arg}"
    elif [[ "${arg}" == arm* ]]; then
      ARCH="${arg}"
    fi
  done

  echo "building edgesite release:" ${VERSION} "ARCH:"${ARCH}

  mkdir -p _output/release/${VERSION}/edgesite-${VERSION}-linux-${ARCH}/edgesite

  echo ${VERSION} > _output/release/${VERSION}/edgesite-${VERSION}-linux-${ARCH}/version
  cp _output/local/bin/edgesite-agent _output/release/${VERSION}/edgesite-${VERSION}-linux-${ARCH}/edgesite
  cp _output/local/bin/edgesite-server _output/release/${VERSION}/edgesite-${VERSION}-linux-${ARCH}/edgesite

  cd _output/release/${VERSION}
  tar -czvf ${KUBEEDGE_ROOT}/_output/release/${VERSION}/edgesite-${VERSION}-linux-${ARCH}.tar.gz edgesite-${VERSION}-linux-${ARCH}/

  cd $KUBEEDGE_ROOT
  rm -r _output/release/${VERSION}/edgesite-${VERSION}-linux-${ARCH}

  #calculate sha512sum
  sum=$(sha512sum _output/release/${VERSION}/edgesite-${VERSION}-linux-${ARCH}.tar.gz)
  sumArray=($sum)
  echo ${sumArray[0]} > _output/release/${VERSION}/checksum_edgesite-${VERSION}-linux-${ARCH}.tar.gz.txt
}

release "$@"
