SOURCES ?= $(shell find . -path "./vendor" -prune -o -type f -name "*.go" -print)

TEST_DB_PORT := 3100

traQ: $(SOURCES)
	CGO_ENABLED=0 go build

.PHONY: init
init:
	go mod download

.PHONY: genkey
genkey:
	mkdir -p ./dev/keys
	cd ./dev/keys && go run ../bin/gen_ec_pem.go

.PHONY: test
test:
	MARIADB_PORT=$(TEST_DB_PORT) go test ./... -race

.PHONY: up-test-db
up-test-db:
	@TEST_DB_PORT=$(TEST_DB_PORT) ./dev/bin/up-test-db.sh

.PHONY: rm-test-db
rm-test-db:
	@./dev/bin/down-test-db.sh

.PHONY: lint
lint:
	-@make golangci-lint
	-@make swagger-lint

.PHONY: golangci-lint
golangci-lint:
	golangci-lint run

.PHONY: swagger-lint
swagger-lint:
	spectral lint -q docs/*.yaml

.PHONY: db-gen-docs
db-gen-docs:
	@if [ -d "./docs/dbschema" ]; then \
		rm -r ./docs/dbschema; \
	fi
	go run main.go migrate --reset --port $(TEST_DB_PORT)
	TBLS_DSN="mysql://root:password@127.0.0.1:$(TEST_DB_PORT)/traq" tbls doc

.PHONY: db-diff-docs
db-diff-docs:
	go run main.go migrate --reset --port $(TEST_DB_PORT)
	TBLS_DSN="mysql://root:password@127.0.0.1:$(TEST_DB_PORT)/traq" tbls diff

.PHONY: db-lint
db-lint:
	go run main.go migrate --reset --port $(TEST_DB_PORT)
	TBLS_DSN="mysql://root:password@127.0.0.1:$(TEST_DB_PORT)/traq" tbls lint

.PHONY: goreleaser-snapshot
goreleaser-snapshot:
	goreleaser --snapshot --skip-publish --rm-dist

.PHONY: update-frontend
update-frontend:
	@mkdir -p ./dev/frontend
	@curl -L -Ss https://github.com/traPtitech/traQ_R-UI/releases/latest/download/dist.tar.gz | tar zxv -C ./dev/frontend/ --strip-components=2

.PHONY: reset-frontend
reset-frontend:
	@if [ -d "./dev/frontend" ]; then \
		rm -r ./dev/frontend; \
	fi
	@make update-frontend

.PHONY: up
up:
	@docker-compose up -d --build

.PHONY: down
down:
	@docker-compose down -v
