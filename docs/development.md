# Development

If you want to contribute to traQ, then follow this section.

### Requirements

- Go 1.27
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

### S3 storage (RustFS)

`make up` starts RustFS and creates the `traq` bucket with the development
credentials in `compose.yaml`. The S3 endpoint is `http://localhost:9000`
(`http://s3:9000` inside Compose), the web console is `http://localhost:9001`,
and the region is `ap-northeast-1`. Persistent data is stored in the `rustfs`
volume. Run storage tests with `go test ./utils/storage`.

#### Existing Garage data

The Garage volume cannot be reused by RustFS. Complete the copy before replacing
the old Garage deployment with this revision, which no longer contains Garage's
runtime configuration:

1. In the old revision, stop only the backend (`docker compose stop backend`) and
   back up the database and Garage volume. Keep Garage running as the source.
2. Provision RustFS at a separate endpoint and create its `traq` bucket.
3. Configure [rclone S3 remotes](https://rclone.org/s3/) named `garage` and `rustfs`
   with their explicit endpoints, region `ap-northeast-1`, and S3 provider `Other`.
   Set `force_path_style = true` for the RustFS remote.
4. Run `rclone copy garage:traq rustfs:traq --metadata`, followed by
   `rclone check garage:traq rustfs:traq --download`.
5. After verification succeeds, deploy this revision, point traQ at RustFS, start
   the backend, and check existing attachments. Keep the Garage backup until the
   migration is confirmed.
