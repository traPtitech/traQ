# Development

If you want to contribute to traQ, then follow this section.

### Requirements

- Go 1.19
- git
- bash
- make
- Docker
- docker-compose

### Setup Local Server with Docker

#### First Up (or entirely rebuild)
`make up`

Now you can access to
+ `http://localhost:3000` for traQ
    + admin user id: `traq`
    + admin user password: `traq`
+ `http://localhost:3001` for Adminer
+ `http://localhost:6060` for traQ pprof web interface
+ `3002/tcp` for traQ MariaDB
    + username: `root`
    + password: `password`
    + database: `traq`

#### Rebuild traQ
`make up`

#### Destroy Containers
`make down`

#### Remove dev data
1. `make down`
2. Remove respective directory in `./dev/data` (e.g. to remove all `rm -r ./dev/data/*`)
3. `make up`

#### Build executable file
`make traQ`

#### Download and Install go mod dependencies
`make init`
> `github.com/google/wire/cmd/wire` and `github.com/golang/mock/mockgen` will be installed.

#### Rerun automated code generation (wire, gomock)
`make gogen`

#### Testing
1. Setup test DB container by `make up-test-db`
2. `make test`
3. (Remove test DB container by `make rm-test-db`)

#### Code Lint
`make lint` (or individually `make golangci-lint`, `make swagger-lint`)

Powered by:
+ [golangci-lint](https://github.com/golangci/golangci-lint) for go codes (pre-installation required)
+ [spectral](https://github.com/stoplightio/spectral) for swagger specs

#### Generate and Lint DB Schema Docs
If your changelist alters the database schema, you should regenerate db docs.

1. Write new schema descriptions in `.tbls.yml`.
2. Make sure the Test DB Container is running (run `make up-test-db`).
3. `make db-gen-docs`

Powered by:
+ [tbls](https://github.com/k1LoW/tbls) for generating schema docs
