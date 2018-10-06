BINARY = prometheus-msteams
VET_REPORT = vet.report
TEST_REPORT = tests.xml
GOARCH = amd64
BINDIR = bin
VERSION?=latest
COMMIT=$(shell git rev-parse --short HEAD)
BRANCH=$(shell git rev-parse --abbrev-ref HEAD)
BUILD_DATE=$(shell date +%FT%T%z)

# Symlink into GOPATH
GITHUB_USERNAME=bzon
BUILD_DIR=$(GOPATH)/src/github.com/$(GITHUB_USERNAME)/$(BINARY)
VERSION_PKG=github.com/bzon/prometheus-msteams/cmd

# Setup the -ldflags option for go build here, interpolate the variable values
LDFLAGS = -ldflags "-X $(VERSION_PKG).version=$(VERSION) -X $(VERSION_PKG).commit=$(COMMIT) -X $(VERSION_PKG).branch=$(BRANCH) -X $(VERSION_PKG).buildDate=$(BUILD_DATE)"

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
	CGO_ENABLED=0 GOOS=linux GOARCH=$(GOARCH) go build $(LDFLAGS) -o $(BINDIR)/$(BINARY)-linux-$(GOARCH) . 

darwin:
	CGO_ENABLED=0 GOOS=darwin GOARCH=$(GOARCH) go build $(LDFLAGS) -o $(BINDIR)/$(BINARY)-darwin-$(GOARCH) .

windows:
	go get -v github.com/konsorten/go-windows-terminal-sequences
	CGO_ENABLED=0 GOOS=windows GOARCH=$(GOARCH) go build $(LDFLAGS) -o $(BINDIR)/$(BINARY)-windows-$(GOARCH).exe . 

docker: clean dep linux
	docker build -t $(GITHUB_USERNAME)/$(BINARY):$(VERSION) .

test-docker-run: docker
	docker run --rm $(DOCKER_RUN_OPTS) -p 2000:2000 $(GITHUB_USERNAME)/$(BINARY):$(VERSION) $(DOCKER_RUN_ARG)

docker-push: docker
	docker push $(GITHUB_USERNAME)/$(BINARY):$(VERSION)

run-osx: dep darwin
	bin/prometheus-msteams-darwin-amd64 server $(RUN_ARGS)

test:
	go test ./... -v -race

coverage:
	go test ./... -v -race -coverprofile=coverage.txt -covermode=atomic

dep:
	go get -v ./...


clean:
	-rm -fr $(BINDIR)
	-rm -f $(BINARY)-*
