# traQ (Project R)

[![GitHub release](https://img.shields.io/github/release/traPtitech/traQ.svg)](https://GitHub.com/traPtitech/traQ/releases/)
![CI](https://github.com/traPtitech/traQ/workflows/CI/badge.svg)
![release](https://github.com/traPtitech/traQ/workflows/release/badge.svg)
[![codecov](https://codecov.io/gh/traPtitech/traQ/branch/master/graph/badge.svg)](https://codecov.io/gh/traPtitech/traQ)
[![Dependabot Status](https://api.dependabot.com/badges/status?host=github&repo=traPtitech/traQ)](https://dependabot.com)


## Development environment

### Requirements

- go 1.13.x
- git
- make
- docker
- docker-compose

### Setup with docker and docker-compose

#### First Up (or entirely rebuild)
`make update-frontend && docker-compose up -d --build`

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
`docker-compose up -d --no-deps --build backend`

#### Update frontend
`make update-frontend`

#### Destroy Containers and Volumes
`docker-compose down -v`

### Testing
1. Run mysql container for test by `make up-test-db`
2. `make test`

You can remove the container by `make rm-test-db`

### Code Lint
`make lint` (or individually `make golangci-lint`, `make swagger-lint`)

Installing below tools in advance is required:
+ [golangci-lint](https://github.com/golangci/golangci-lint) for go codes
+ [spectral](https://github.com/stoplightio/spectral) for swagger specs

### Generate DB Schema Docs
[tbls](https://github.com/k1LoW/tbls) is required.

`make db-gen-docs`

Test mysql container need to be running by `make up-test-db`.

#### DB Docs Lint
`make db-lint`

## License
Code licensed under [the MIT License](https://github.com/traPtitech/traQ/blob/master/LICENSE).

[twemoji](https://twemoji.twitter.com) (svg files in `/dev/data/twemoji`) by 2018 Twitter, Inc and other contributors is licensed under [CC-BY 4.0](https://creativecommons.org/licenses/by/4.0/). 
