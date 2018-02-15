SOURCES ?= $(shell find . -path "./vendor" -prune -o -type f -name "*.go" -print)

traQ: $(SOURCES)
	go build

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: test
test:
	gofmt -s -d $(SOURCES)
	-golint -set_exit_status
	-go vet ./...
	-go test -race ./...
	-go build

.PHONY: init
init:
	go get -u github.com/golang/dep/cmd/dep
	go get -u github.com/golang/lint/golint
	dep ensure
