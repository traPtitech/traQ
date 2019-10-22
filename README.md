# traQ (Project R)

[![CircleCI](https://circleci.com/gh/traPtitech/traQ.svg?style=shield)](https://circleci.com/gh/traPtitech/traQ)
[![codecov](https://codecov.io/gh/traPtitech/traQ/branch/master/graph/badge.svg)](https://codecov.io/gh/traPtitech/traQ)
[![Dependabot Status](https://api.dependabot.com/badges/status?host=github&repo=traPtitech/traQ)](https://dependabot.com)


## Development environment

### Requirements

- go 1.13.x
- git
- make
- openssl
- docker
- docker-compose

### Setup with docker and docker-compose

#### First Up (or entirely rebuild)
`make init && docker-compose up -d --build`

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
`docker-compose up -d --no-deps --build traq-backend`

#### Destroy Containers and Volumes
`docker-compose down -v`

## License
Code licensed under [the MIT License](https://github.com/traPtitech/traQ/blob/master/LICENSE).

[twemoji](https://twemoji.twitter.com) (svg files in `/dev/data/twemoji`) by 2018 Twitter, Inc and other contributors is licensed under [CC-BY 4.0](https://creativecommons.org/licenses/by/4.0/). 
