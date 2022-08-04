
BINARY = crow
VET_REPORT = vet.report
TEST_REPORT = tests.xml
GOARCH = amd64

VERSION?=?
COMMIT=$(shell git rev-parse --short HEAD)
BRANCH=$(shell git rev-parse --abbrev-ref HEAD)

# Symlink into GOPATH
DOCKERHUB_USERNAME=ab-ty
BUILD_DIR=./bin
CURRENT_DIR=$(shell pwd)
BUILD_DIR_LINK=$(shell readlink ${BUILD_DIR})
DATE=$(shell date +'%Y%m%d%M%S')
REPO_NAME?=${DOCKERHUB_USERNAME}/${BINARY}
# Setup the -ldflags option for go build here, interpolate the variable values
LDFLAGS = -ldflags "-X main.VERSION=${VERSION} -X main.COMMIT=${COMMIT} -X main.BRANCH=${BRANCH}"

# Build the project
all: link clean test vet linux image-build

linux:
	cd ${BUILD_DIR}; \
	GOOS=linux GOARCH=${GOARCH} go build ${LDFLAGS} -o ${BINARY}-linux-${GOARCH} . ; \
	cd - >/dev/null

image-build:
	docker build . --file Dockerfile --tag ${BINARY}:$(COMMIT)

image-push: image-build
	docker tag ${BINARY}:${COMMIT} ${REPO_NAME}:${COMMIT}
	docker tag ${BINARY}:${COMMIT} ${REPO_NAME}:latest

	echo "${DOCCKER_TOKEN}" | docker login -u ${DOCKER_USERNAME} --password-stdin
	docker push ${REPO_NAME}:${COMMIT}
	docker push ${REPO_NAME}:latest



test:
	go test -v ./...

vet:
	go vet ./...

fmt:
	go fmt $$(go list ./... | grep -v /vendor/)

clean:
	-rm -f ${TEST_REPORT}
	-rm -f ${VET_REPORT}
	-rm -f ${BINARY}-*

.PHONY: link linux darwin windows test vet fmt clean