# traQ (Project R)

[![CircleCI](https://circleci.com/gh/traPtitech/traQ.svg?style=shield)](https://circleci.com/gh/traPtitech/traQ)
[![codecov](https://codecov.io/gh/traPtitech/traQ/branch/master/graph/badge.svg)](https://codecov.io/gh/traPtitech/traQ)

## Development environment

### Requirements

- go
	- tested with 1.11
- git
- make

### Setup with docker and docker-compose (Recommended)

#### First Up (or entirely rebuild)
`docker-compose up -d --build`

Now you can access to
+ `http://localhost:3000` for traQ
+ `http://localhost:3001` for Adminer(Browser Database Management Tool)
+ `http://localhost:6060` for traQ pprof web interface
+ `3002/tcp` for traQ MariaDB
    + username: `root`
    + password: `password`
    + database: `traq`

#### Rebuild traQ
`docker-compose up -d --no-deps --build traq-backend`

#### Destroy Containers and Volumes
`docker-compose down -v`

### Setup (for Linux, macOS)

Setup [GOPATH](https://github.com/golang/go/wiki/GOPATH) first

Set Environment Variable 'GO111MODULE' to 'on'.
We recommend using [direnv](https://github.com/direnv/direnv) for setting up it.

```
make init
make
```

### Setup with Vagrant (recommended for Windows)

Use [Vagrant](https://www.vagrantup.com/downloads.html)

```
vagrant plugin install vagrant-itamae
vagrant up
```

```
vagrant ssh
make init
make
```
