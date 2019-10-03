PROJECT_NAME=bonus_manger
VERSION = 0.0.1
BUILD_ENV := CGO_ENABLED=0

LDFLAGS=-ldflags "-s -w"
TARGET_EXEC = bonus_manger
GO_FILE = $(ls *.go|grep -v _test)
SOC:=$(shell uname -m)

.PHONY: all

all: setup build-amd64 build-aarch64 build-arm
	md5sum `find build -name 'bon*'`>build/md5sum

setup:
	mkdir -p build/


build-aarch64: setup
	${BUILD_ENV} GOARCH=arm64 GOOS=linux  go build ${LDFLAGS} -o build/${TARGET_EXEC}_aarch64 ${GO_FILE}

build-arm: setup
	${BUILD_ENV} GOARCH=arm GOOS=linux go build ${LDFLAGS} -o build/${TARGET_EXEC}_armv7l ${GO_FILE}

build-amd64: setup
	${BUILD_ENV} GOARCH=amd64 GOOS=linux go build ${LDFLAGS} -o build/${TARGET_EXEC}_x86_64 ${GO_FILE}

docker:  docker-aarch64 docker-amd64

docker-amd64: build-amd64
	docker build -f 'docker/Dockerfile' -t  bonusmanger:latest .

docker-aarch64: build-aarch64
	docker -H tcp://node1.lan build -f 'docker/Dockerfile_aarch64' -t  bonusmanger:latest .



clean:
	rm -rf build
