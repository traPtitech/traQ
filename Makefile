SOURCES ?= $(shell find . -path "./vendor" -prune -o -type f -name "*.go" -print)

traQ: $(SOURCES)
	go build

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: lint
lint:
	-@go vet $$(glide novendor)
	-@for pkg in $$(glide novendor -x); do golint -set_exit_status $$pkg; done

.PHONY: test
test:
	gofmt -s -d $(SOURCES)
	-@make lint
	-go test -race ./...
	-go build

.PHONY: init
init:
	go get -u github.com/golang/dep/cmd/dep
	go get -u github.com/golang/lint/golint
	go get -u github.com/Masterminds/glide
	dep ensure
