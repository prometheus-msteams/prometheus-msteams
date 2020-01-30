BINARY = prometheus-msteams
VET_REPORT = vet.report
TEST_REPORT = tests.xml
GOARCH = amd64
BINDIR = bin
VERSION?=latest
COMMIT=$(shell git rev-parse --short HEAD)
BRANCH=$(shell git rev-parse --abbrev-ref HEAD)
BUILD_DATE=$(shell date +%FT%T%z)
GOFMT_FILES?=$$(find . -name '*.go')
GO := GO111MODULE=on go

# Symlink into GOPATH
GITHUB_USERNAME=bzon
BUILD_DIR=$(GOPATH)/src/github.com/$(GITHUB_USERNAME)/$(BINARY)
VERSION_PKG=github.com/bzon/prometheus-msteams/pkg/version

# Setup the -ldflags option for go build here, interpolate the variable values
LDFLAGS = -ldflags "-X $(VERSION_PKG).VERSION=$(VERSION) -X $(VERSION_PKG).COMMIT=$(COMMIT) -X $(VERSION_PKG).BRANCH=$(BRANCH) -X $(VERSION_PKG).BUILDDATE=$(BUILD_DATE)"

DOCKER_RUN_OPTS ?=
DOCKER_RUN_ARG ?=

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

docker: clean dep linux
	docker build -t $(GITHUB_USERNAME)/$(BINARY):$(VERSION) .

test-docker-run: docker
	docker run --rm $(DOCKER_RUN_OPTS) -p 2000:2000 $(GITHUB_USERNAME)/$(BINARY):$(VERSION) $(DOCKER_RUN_ARG)

docker-push: docker
	docker push $(GITHUB_USERNAME)/$(BINARY):$(VERSION)

run-osx: dep darwin
	bin/prometheus-msteams-darwin-amd64 server $(RUN_ARGS)

fmt:
	gofmt -w $(GOFMT_FILES)

lint:
	golint -set_exit_status ./...

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
