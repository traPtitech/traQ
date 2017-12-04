sources = `find . -path "./vendor" -prune -o -type f -name "*.go" -print`

.PHONY: test
test:
	gofmt -s -l -w $(sources)
	-golint -set_exit_status
	-go vet ./...
	-go test ./...
	-go build

.PHONY: init
init:
	go get -u github.com/golang/dep/cmd/dep
	go get -u github.com/golang/lint/golint
	dep ensure
