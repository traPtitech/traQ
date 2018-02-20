SOURCES ?= $(shell find . -path "./vendor" -prune -o -type f -name "*.go" -print)

traQ: $(SOURCES)
	go build

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: test
test:
	-@make ci-fmt
	-@make ci-vet
	-@make ci-lint
	-@make ci-test
	-@make traQ

.PHONY: ci-fmt
ci-fmt:
	(! gofmt -s -d `find . -path "./vendor" -prune -o -type f -name "*.go" -print` | grep ^)

.PHONY: ci-vet
ci-vet:
	go vet ./...

.PHONY: ci-lint
ci-lint:
	golint -set_exit_status $$(go list ./...)

.PHONY: ci-test
ci-test:
	go test -race ./...

.PHONY: init
init:
	go get -u github.com/golang/dep/cmd/dep
	go get -u github.com/golang/lint/golint
	dep ensure
