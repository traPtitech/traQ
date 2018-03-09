# traQ (Project R)

[![CircleCI](https://circleci.com/gh/traPtitech/traQ.svg?style=shield)](https://circleci.com/gh/traPtitech/traQ)
[![codecov](https://codecov.io/gh/traPtitech/traQ/branch/master/graph/badge.svg)](https://codecov.io/gh/traPtitech/traQ)
[![Heroku](https://heroku-badge.herokuapp.com/?app=traq-dev&svg=1)](https://traq-dev.herokuapp.com/)

## Development environment

### Requirements

- go
	- tested with 1.10
- git
- make

### Setup (for Linux, macOS)

Setup [GOPATH](https://github.com/golang/go/wiki/GOPATH) first

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
