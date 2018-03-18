BINARY = prometheus-msteams
VET_REPORT = vet.report
TEST_REPORT = tests.xml
GOARCH = amd64

VERSION?=latest
COMMIT=$(shell git rev-parse HEAD)
BRANCH=$(shell git rev-parse --abbrev-ref HEAD)

# Symlink into GOPATH
GITHUB_USERNAME=bzon
BUILD_DIR=${GOPATH}/src/github.com/${GITHUB_USERNAME}/${BINARY}

# Setup the -ldflags option for go build here, interpolate the variable values
LDFLAGS = -ldflags "-X main.VERSION=${VERSION} -X main.COMMIT=${COMMIT} -X main.BRANCH=${BRANCH}"

# Build the project
all: clean getdep linux darwin windows

linux: 
	echo Build for linux ${GOARCH}
	GOOS=linux GOARCH=${GOARCH} go build ${LDFLAGS} -o ${BINARY}-linux-${GOARCH} . 

darwin:
	echo Build for darwin ${GOARCH}
	GOOS=darwin GOARCH=${GOARCH} go build ${LDFLAGS} -o ${BINARY}-darwin-${GOARCH} .

windows:
	echo Build for windows ${GOARCH}
	GOOS=windows GOARCH=${GOARCH} go build ${LDFLAGS} -o ${BINARY}-windows-${GOARCH}.exe . 

docker: clean getdep test linux
	echo Performing a docker build
	docker build --build-arg version=${VERSION} -t ${GITHUB_USERNAME}/${BINARY}:${VERSION} .

docker_push: docker
	docker push ${GITHUB_USERNAME}/${BINARY}:${VERSION}

test:
	echo Performing a go test
	go test ./... -v

getdep:
	go get -v ./...

vet:
	godep go vet ./... > ${VET_REPORT} 2>&1

fmt:
	go fmt $$(go list ./... | grep -v /vendor/)

clean:
	-rm -f ${TEST_REPORT}
	-rm -f ${VET_REPORT}
	-rm -f ${BINARY}-*