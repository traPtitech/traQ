# traQ - traP Internal Messenger Application

[![GitHub release](https://img.shields.io/github/release/traPtitech/traQ.svg)](https://GitHub.com/traPtitech/traQ/releases/)
![CI](https://github.com/traPtitech/traQ/workflows/CI/badge.svg)
![release](https://github.com/traPtitech/traQ/workflows/release/badge.svg)
[![codecov](https://codecov.io/gh/traPtitech/traQ/branch/master/graph/badge.svg)](https://codecov.io/gh/traPtitech/traQ)
[![Dependabot Status](https://api.dependabot.com/badges/status?host=github&repo=traPtitech/traQ)](https://dependabot.com)
[![swagger](https://img.shields.io/badge/swagger-docs-brightgreen)](https://traptitech.github.io/traQ/)

Backend: this repository

Frontend: [traQ_S-UI](https://github.com/traPtitech/traQ_S-UI)

## Development environment

### Requirements

- go 1.14
- git
- bash
- make
- docker
- docker-compose

### Setup Local Server with docker

#### First Up (or entirely rebuild)
`make update-frontend && make up`

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

#### Update frontend
`make update-frontend` or `make reset-frontend`

#### Destroy Containers and Volumes
`make down`

### Development

#### Build executable file
`make`

#### Download and Install go mod dependencies
`make init`
> `github.com/google/wire/cmd/wire` and `github.com/golang/mock/mockgen` will be installed.

#### Rerun automated code generation (wire, gomock)
`make gogen`

#### Testing
1. To run mysql container for test by `make up-test-db`
2. `make test`

You can remove the container by `make rm-test-db`

#### Code Lint
`make lint` (or individually `make golangci-lint`, `make swagger-lint`)

Powered by:
+ [golangci-lint](https://github.com/golangci/golangci-lint) for go codes (pre-installation required)
+ [spectral](https://github.com/stoplightio/spectral) for swagger specs

#### Generate And Lint DB Schema Docs
If your code changes alter database schema, you should regenerate db docs.

1. Write new schema descriptions in `.tbls.yml`.
2. Make sure that Test MySQL Container are running (Do `make up-test-db`).
3. `make db-gen-docs`

[tbls](https://github.com/k1LoW/tbls) is used for generating schema docs.

## License
Code licensed under [the MIT License](https://github.com/traPtitech/traQ/blob/master/LICENSE).

This application uses [twemoji](https://twemoji.twitter.com)'s SVG images as Unicode emoji stamps.
[twemoji](https://twemoji.twitter.com) by 2019 Twitter, Inc and other contributors is licensed under [CC-BY 4.0](https://creativecommons.org/licenses/by/4.0/). 
