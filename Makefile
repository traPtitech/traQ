sources = `find -path "./vendor" -prune -o -type f -name "*.go" -print`

.PHONY: test init

test:
	gofmt -s -l -w $(sources)
	-golint -set_exit_status
	-go vet
	-go build

init:
	go get -u github.com/golang/dep/cmd/dep
	go get -u github.com/golang/lint/golint
	dep ensure
