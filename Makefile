BUILD_VERSION ?= v0.0.1
BUILD_CONTAINER ?= spremkumar/gopher-builder:v1.0
OUT_CONTAINER ?=  spremkumar/gandalf:${BUILD_VERSION}
GIT_COMMIT ?= $(shell git rev-list -1 HEAD --abbrev-commit)
UID ?= $(shell id -u)
GID ?= $(shell id -g)
DIRS=$(shell go list -f {{.Dir}} github.com/supriya-premkumar/gandalf/...)
GOIMPORTS_CMD := goimports -local "github.com/supriya-premkumar/gandalf" -l -e

.PHONY: compile
compile:
	docker run --rm -u ${UID}:${GID} -e GOCACHE=/go/src/github.com/supriya-premkumar/gandalf/.cache \
                                     -v ${PWD}:/go/src/github.com/supriya-premkumar/gandalf/ \
                                     ${BUILD_CONTAINER} bash -c "make -C /go/src/github.com/supriya-premkumar/gandalf build"

container:
	$(info +++ building gandalf images  ${OUT_CONTAINER})
	cp bin/gandalf.linux deploy/gandalf
	docker build -f deploy/Dockerfile --build-arg GIT_COMMIT=${GIT_COMMIT} -t ${OUT_CONTAINER} .

build:
	$(info +++ goimports sources)
	@${GOIMPORTS_CMD} -w ${DIRS}

	$(info +++ linting sources)
	@eval 'revive -config /go/revive.toml --formatter friendly --exclude ./vendor/... ./...'

	$(info +++ vetting sources)
	@go vet ./...

	$(info +++ building sources)
	go mod tidy
	CGO_ENABLED=0 GOARCH=amd64 GOOS=darwin go build -ldflags "-X main.GitCommit=${GIT_COMMIT}" \
	-o bin/gandalf.darwin github.com/supriya-premkumar/gandalf
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -ldflags "-X main.GitCommit=${GIT_COMMIT}" \
	-o bin/gandalf.linux github.com/supriya-premkumar/gandalf
