SOURCES ?= $(shell find . -type f \( -name "*.go" -o -name "go.mod" -o -name "go.sum" \) -print)

TEST_DB_PORT := 3100
# renovate:image-tag imageName=ghcr.io/k1low/tbls
TBLS_VERSION := "v1.74.2"
# renovate:image-tag imageName=index.docker.io/stoplight/spectral
SPECTRAL_VERSION := "6.11.1"

.DEFAULT_GOAL := help

.PHONY: help
help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

traQ: $(SOURCES) ## Build traQ binary
	CGO_ENABLED=0 go build -o traQ -ldflags "-s -w -X main.version=Dev -X main.revision=Local"

.PHONY: init
init: ## Download and install go mod dependencies
	go mod download
	go install github.com/google/wire/cmd/wire@v0.6.0
	go install github.com/golang/mock/mockgen@v1.6.0

.PHONY: genkey
genkey: ## Generate dev keys
	mkdir -p ./dev/keys
	cd ./dev/keys && go run ../bin/gen_ec_pem.go

.PHONY: test
test: ## Run test
	MARIADB_PORT=$(TEST_DB_PORT) go test ./... -race -shuffle=on

.PHONY: up-test-db
up-test-db: ## Make sure the test db container is running
	@TEST_DB_PORT=$(TEST_DB_PORT) ./dev/bin/up-test-db.sh

.PHONY: rm-test-db
rm-test-db: ## Remove the test db container
	@./dev/bin/down-test-db.sh

.PHONY: lint
lint: ## Lint go and swagger files
	-@make golangci-lint
	-@make swagger-lint

.PHONY: golangci-lint
golangci-lint: ## Lint go files
	@golangci-lint run

.PHONY: swagger-lint
swagger-lint: ## Lint swagger file
	@docker run --rm -it -v $$PWD:/tmp stoplight/spectral:$(SPECTRAL_VERSION) lint -r /tmp/.spectral.yml -q /tmp/docs/v3-api.yaml

.PHONY: db-gen-docs
db-gen-docs: ## Generate db docs in docs/dbSchema
	TRAQ_MARIADB_PORT=$(TEST_DB_PORT) go run main.go migrate --reset
	docker run --rm --net=host -e TBLS_DSN="mariadb://root:password@127.0.0.1:$(TEST_DB_PORT)/traq" -v $$PWD:/work -w /work ghcr.io/k1low/tbls:$(TBLS_VERSION) doc -c .tbls.yml --rm-dist

.PHONY: db-diff-docs
db-diff-docs: ## List diff of db docs
	TRAQ_MARIADB_PORT=$(TEST_DB_PORT) go run main.go migrate --reset
	docker run --rm --net=host -e TBLS_DSN="mariadb://root:password@127.0.0.1:$(TEST_DB_PORT)/traq" -v $$PWD:/work -w /work ghcr.io/k1low/tbls:$(TBLS_VERSION) diff -c .tbls.yml

.PHONY: db-lint
db-lint: ## Lint db docs according to .tbls.yml
	TRAQ_MARIADB_PORT=$(TEST_DB_PORT) go run main.go migrate --reset
	docker run --rm --net=host -e TBLS_DSN="mariadb://root:password@127.0.0.1:$(TEST_DB_PORT)/traq" -v $$PWD:/work -w /work ghcr.io/k1low/tbls:$(TBLS_VERSION) lint -c .tbls.yml

.PHONY: goreleaser-snapshot
goreleaser-snapshot: ## Release dry-run
	@docker run --rm -it -v $$PWD:/src -w /src goreleaser/goreleaser --snapshot --skip-publish --rm-dist

.PHONY: update-frontend
update-frontend: ## Update frontend files in dev/frontend
	@mkdir -p ./dev/frontend
# renovate:github-url
	@curl -L -Ss https://github.com/traPtitech/traQ_S-UI/releases/download/v3.19.1/dist.tar.gz | tar zxv -C ./dev/frontend/ --strip-components=2

.PHONY: reset-frontend
reset-frontend: ## Completely replace frontend files in dev/frontend
	rm -rf ./dev/frontend
	@make update-frontend

.PHONY: up
up: ## Build and start the app containers
	@docker compose up -d --build

.PHONY: down
down: ## Stop and remove app containers
	@docker compose down

.PHONY: gogen
gogen: ## Generate auto-generated go files
	go generate ./...
