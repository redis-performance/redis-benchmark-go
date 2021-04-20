# Go parameters
GOCMD=GO111MODULE=on go
GOBUILD=$(GOCMD) build
GOINSTALL=$(GOCMD) install
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
DISTDIR = ./dist

.PHONY: all test coverage
all: test coverage build

build:
	$(GOBUILD) .

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

test: get
	$(GOFMT) ./...
	$(GOTEST) -race -covermode=atomic ./...

coverage: get test
	$(GOTEST) -race -coverprofile=coverage.txt -covermode=atomic .


release:
	$(GOGET) github.com/mitchellh/gox
	$(GOGET) github.com/tcnksm/ghr
	GO111MODULE=on gox  -osarch "linux/amd64 linux/arm64 windows/amd64 darwin/amd64 darwin/arm64" -output "${DISTDIR}/redis-benchmark-go_{{.OS}}_{{.Arch}}" .

publish: release
	@for f in $(shell ls ${DISTDIR}); \
	do \
	echo "copying ${DISTDIR}/$${f}"; \
	aws s3 cp ${DISTDIR}/$${f} s3://benchmarks.redislabs/tools/redis-benchmark-go/$${f} --acl public-read; \
	done

fmt:
	$(GOFMT) ./...

