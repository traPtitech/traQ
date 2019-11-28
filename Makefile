SOURCES ?= $(shell find . -path "./vendor" -prune -o -type f -name "*.go" -print)

traQ: $(SOURCES)
	go build -ldflags "-X main.version=$$(git describe --tags --abbrev=0) -X main.revision=$$(git rev-parse --short HEAD)"

.PHONY: init
init:
	go mod download

.PHONY: genkey
genkey:
	mkdir -p ./dev/keys
	cd ./dev/keys && go run ../bin/gen_ec_pem.go

.PHONY: test
test:
	MARIADB_PORT=3100 go test ./... -race

.PHONY: up-test-db
up-test-db:
	@./dev/bin/up-test-db.sh

.PHONY: rm-test-db
rm-test-db:
	@./dev/bin/down-test-db.sh

.PHONY: make-db-docs
make-db-docs:
	if [ -d "./docs/dbschema" ]; then \
		rm -r ./docs/dbschema; \
	fi
	TBLS_DSN="mysql://root:password@127.0.0.1:3002/traq" tbls doc

.PHONY: diff-db-docs
diff-db-docs:
	TBLS_DSN="mysql://root:password@127.0.0.1:3002/traq" tbls diff
