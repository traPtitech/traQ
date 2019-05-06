SOURCES ?= $(shell find . -path "./vendor" -prune -o -type f -name "*.go" -print)

traQ: $(SOURCES)
	go build -ldflags "-X main.version=$$(git describe --tags --abbrev=0) -X main.revision=$$(git rev-parse --short HEAD)"

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: test
test:
	-@make ci-fmt
	-@make ci-vet
	-@make ci-lint
	-@make ci-test

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
	go mod download
	go install golang.org/x/lint/golint
	mkdir -p ./keys
	openssl ecparam -genkey -name prime256v1 -noout -out ec.pem
	openssl ec -in ec.pem -out ec_pub.pem -pubout

.PHONY: up-docker-test-db
up-docker-test-db:
	docker run --name traq-test-db -p 3000:3306 -e MYSQL_ROOT_PASSWORD=password -d mariadb:10.0.19 mysqld --character-set-server=utf8mb4 --collation-server=utf8mb4_general_ci
	sleep 5
	TEST_DB_PORT=3000 go run .circleci/init.go

.PHONY: start-docker-test-db
start-docker-test-db:
	docker start traq-test-db

.PHONY: stop-docker-test-db
stop-docker-test-db:
	docker stop traq-test-db

.PHONY: down-docker-test-db
down-docker-test-db:
	docker rm -f -v traq-test-db
