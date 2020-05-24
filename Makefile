BINARY = prometheus-msteams
VET_REPORT = vet.report
TEST_REPORT = tests.xml
GOARCH = amd64
BINDIR = bin
VERSION:=$(shell git describe --tags --always --dirty)
COMMIT=$(shell git rev-parse --short HEAD)
BRANCH=$(shell git rev-parse --abbrev-ref HEAD)
BUILD_DATE=$(shell date +%FT%T%z)
GOFMT_FILES?=$$(find . -name '*.go')
GO := GO111MODULE=on go

# Symlink into GOPATH
GITHUB_USERNAME=prometheus-msteams
BUILD_DIR=$(GOPATH)/src/github.com/$(GITHUB_USERNAME)/$(BINARY)
VERSION_PKG=github.com/$(GITHUB_USERNAME)/prometheus-msteams/pkg/version

# Setup the -ldflags option for go build here, interpolate the variable values
LDFLAGS = -ldflags "-X $(VERSION_PKG).VERSION=$(VERSION) -X $(VERSION_PKG).COMMIT=$(COMMIT) -X $(VERSION_PKG).BRANCH=$(BRANCH) -X $(VERSION_PKG).BUILDDATE=$(BUILD_DATE)"

DOCKER_RUN_OPTS ?=
DOCKER_RUN_ARG ?=
RUN_ARGS ?=

# docker
DOCKER_QUAY_REPO=quay.io/prometheusmsteams/prometheus-msteams
DOCKER_QUAY_USER=prometheusmsteams+ci
DOCKER_HUB_REPO=prometheusmsteams/prometheus-msteams

# Build the project
all: clean dep create_bin_dir linux darwin
	cd $(BINDIR) && shasum -a 256 ** > shasum256.txt

create_bin_dir:
	rm -fr $(BINDIR)
	mkdir -p $(BINDIR)

github_release:
	github-release release -u bzon -r prometheus-msteams -t $(VERSION) -n $(VERSION)
	
linux: 
	CGO_ENABLED=0 GOOS=linux GOARCH=$(GOARCH) $(GO) build $(LDFLAGS) -o $(BINDIR)/$(BINARY)-linux-$(GOARCH) ./cmd/server

darwin:
	CGO_ENABLED=0 GOOS=darwin GOARCH=$(GOARCH) $(GO) build $(LDFLAGS) -o $(BINDIR)/$(BINARY)-darwin-$(GOARCH) ./cmd/server

docker:
	docker build -t $(DOCKER_HUB_REPO):$(VERSION) .
	docker tag $(DOCKER_HUB_REPO):$(VERSION) $(DOCKER_QUAY_REPO):$(VERSION)

docker-hub-login:
	echo ${DOCKER_PASSWORD} | docker login --password-stdin -u ${DOCKER_USER}

docker-quay-login:
	docker login -u="${DOCKER_QUAY_USER}" -p="${DOCKER_QUAY_TOKEN}" quay.io

docker-quay-push:
	docker push $(DOCKER_QUAY_REPO):$(VERSION)

docker-tag-latest:
	docker tag $(DOCKER_HUB_REPO):$(VERSION) $(DOCKER_HUB_REPO):latest
	docker tag $(DOCKER_QUAY_REPO):$(VERSION) $(DOCKER_QUAY_REPO):latest

docker-hub-push: docker
	docker push $(DOCKER_HUB_REPO):$(VERSION)

run:
	go run cmd/server/main.go -http-addr=localhost:2000 $(RUN_ARGS)

run-test-config:
	go run cmd/server/main.go -http-addr=localhost:2000 -config-file ./test-connectors.yaml

fmt:
	gofmt -w $(GOFMT_FILES)

lint:
	golangci-lint run ./...

test:
	$(GO) test ./... -v -race

coverage:
	$(GO) test ./... -v -race -coverprofile=coverage.txt -covermode=atomic

dep:
	$(GO) mod tidy
	$(GO) mod download


clean:
	-rm -fr $(BINDIR)
	-rm -f $(BINARY)-*
