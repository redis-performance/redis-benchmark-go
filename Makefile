# Go parameters
GOCMD=GO111MODULE=on go
GOBUILD=$(GOCMD) build
GOBUILDRACE=$(GOCMD) build -race
GOINSTALL=$(GOCMD) install
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
DISTDIR = ./dist
OS_ARCHs = "linux/amd64 linux/arm64 linux/arm windows/amd64 darwin/amd64 darwin/arm64"

# Build-time GIT variables
ifeq ($(GIT_SHA),)
GIT_SHA:=$(shell git rev-parse HEAD)
endif

ifeq ($(GIT_DIRTY),)
GIT_DIRTY:=$(shell git diff --no-ext-diff 2> /dev/null | wc -l)
endif

GCFLAGS:=
ifeq ($(DEBUG),1)
GCFLAGS:=-gcflags="all=-N -l"
endif


.PHONY: all test coverage
all: build

build-coverage:
	$(GOBUILD) -cover \
	-ldflags="-X 'main.GitSHA1=$(GIT_SHA)' -X 'main.GitDirty=$(GIT_DIRTY)'" .

build:
	$(GOBUILD) \
	-ldflags="-X 'main.GitSHA1=$(GIT_SHA)'-X 'main.GitDirty=$(GIT_DIRTY)'"  $(GCFLAGS)  .

build-race:
	$(GOBUILDRACE) \
                   	-ldflags="-X 'main.GitSHA1=$(GIT_SHA)' -X 'main.GitDirty=$(GIT_DIRTY)'" .

checkfmt:
	@echo 'Checking gofmt';\
 	bash -c "diff -u <(echo -n) <(gofmt -d .)";\
	EXIT_CODE=$$?;\
	if [ "$$EXIT_CODE"  -ne 0 ]; then \
		echo '$@: Go files must be formatted with gofmt'; \
	fi && \
	exit $$EXIT_CODE

lint:
	$(GOGET) github.com/golangci/golangci-lint/cmd/golangci-lint
	golangci-lint run

get:
	$(GOGET) -t -v ./...

test: get build-coverage
	@rm -fr .coverdata
	@mkdir -p .coverdata
	@go test -cover -args -test.gocoverdir=".coverdata" .
	@go tool covdata percent -i=.coverdata
	@go tool covdata textfmt -i=.coverdata -o coverage.txt

release:
	$(GOGET) github.com/mitchellh/gox
	$(GOGET) github.com/tcnksm/ghr
	GO111MODULE=on gox  -osarch ${OS_ARCHs} \
	    -ldflags="-X 'main.GitSHA1=$(GIT_SHA)' -X 'main.GitDirty=$(GIT_DIRTY)'" \
	    -output "${DISTDIR}/redis-benchmark-go_{{.OS}}_{{.Arch}}" .

publish: release
	@for f in $(shell ls ${DISTDIR}); \
	do \
	echo "copying ${DISTDIR}/$${f}"; \
	aws s3 cp ${DISTDIR}/$${f} s3://benchmarks.redislabs/tools/redis-benchmark-go/$${f} --acl public-read; \
	done

fmt:
	$(GOFMT) ./...

