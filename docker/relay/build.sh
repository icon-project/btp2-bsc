#!/bin/sh

BASE_DIR=$(dirname $0)
. ${BASE_DIR}/../version.sh

build_image() {
    if [ $# -lt 1 ] ; then
        echo "Usage: $0 <image_name> [<src_dir>] [<build_dir>]"
        return 1
    fi

    local TAG=$1
    local SRC_DIR=$2
    if [ -z "${SRC_DIR}" ] ; then
        SRC_DIR="."
    fi
    local BUILD_DIR=$3

    # Prepare build directory if it's set
    if [ "${BUILD_DIR}" != "" ] ; then
        rm -rf ${BUILD_DIR}
        mkdir -p ${BUILD_DIR}
        cp ${BASE_DIR}/* ${BUILD_DIR}
    else
        BUILD_DIR=${BASE_DIR}
    fi

    BIN_DIR=${BIN_DIR:-${SRC_DIR}/bin}
    if [ "${BUILD_TAGS}" != "" ] ; then
	    BIN_VERSION="${BIN_VERSION}-tags(${BUILD_TAGS})"
    fi

    # copy bin and contracts
    rm -rf ${BUILD_DIR}/dist
    mkdir -p ${BUILD_DIR}/dist/bin/
    cp ${BIN_DIR}/* ${BUILD_DIR}/dist/bin/

    CDIR=$(pwd)
    cd ${BUILD_DIR}

    echo "Building image ${TAG}"
    docker build \
        --build-arg BASE_IMAGE="${BASE_IMAGE:-alpine:${ALPINE_VERSION}}" \
        --build-arg VERSION="${BIN_VERSION}" \
        --tag ${TAG} .
    local result=$?

    cd ${CDIR}
    return $result
}

build_image "$@"
