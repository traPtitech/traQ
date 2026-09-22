# Development

If you want to contribute to traQ, then follow this section.

### Requirements

- git
- bash
- [mise](https://mise.jdx.dev/)
- Docker with the Compose plugin (`docker compose`)

Go and the development tools are installed by mise at the versions specified in
[`mise.toml`](../mise.toml). Running the tests also requires a C compiler for
the Go race detector (`-race`).

### Setup with mise

Use mise to install Go, wire, mockgen, and
golangci-lint at the versions specified in `mise.toml`:

```sh
mise trust
mise install
mise run init
```

Run development tasks with `mise run <task>`, for example `mise run traQ`,
`mise run gogen`, or `mise run lint`. Run `mise run help` to list the tasks.
The mise `init` task downloads Go module dependencies; tools are managed by
`mise install`. Docker with Compose must be installed separately; spectral and
tbls run through Docker.

### Setup Local Server with Docker

#### First Up (or entirely rebuild)
`mise run up`

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
`mise run up`

#### Destroy Containers
`mise run down`

#### Remove dev data
1. `mise run down`
2. Remove respective directory in `./dev/data` (e.g. to remove all `rm -r ./dev/data/*`)
3. `mise run up`

#### Build executable file
`mise run traQ`

#### Download Go module dependencies
`mise run init`

#### Rerun automated code generation (wire, gomock)
`mise run gogen`

#### Testing
1. Setup test DB container by `mise run up-test-db`
2. `mise run test`
3. (Remove test DB container by `mise run rm-test-db`)

#### Code Lint
`mise run lint` (or individually `mise run golangci-lint`, `mise run swagger-lint`)

Powered by:
+ [golangci-lint](https://github.com/golangci/golangci-lint) for go codes (installed by mise)
+ [spectral](https://github.com/stoplightio/spectral) for swagger specs

#### Generate and Lint DB Schema Docs
If your changelist alters the database schema, you should regenerate db docs.

1. Write new schema descriptions in `.tbls.yml`.
2. Make sure the Test DB Container is running (run `mise run up-test-db`).
3. `mise run db-gen-docs`

Powered by:
+ [tbls](https://github.com/k1LoW/tbls) for generating schema docs

### S3 storage (Garage)

`mise run up` starts Garage and automatically creates the `traq` bucket and development
credentials. The S3 endpoint is `http://localhost:9000` (`http://s3:9000` inside
Compose), with region `ap-northeast-1`. Configuration is in `dev/garage.toml`,
credentials in `compose.yaml`, and persistent data in the `garage` volume.
There is no web console; use `docker compose exec s3 /garage status` to check status.
Run storage tests with `go test ./utils/storage`.

Uploads use CRC64NVME checksums for Garage compatibility; other S3 providers must
also support this algorithm.

#### Existing MinIO data

The old `s3` volume cannot be reused by Garage. To migrate:

1. Stop the backend and back up the database and storage volumes.
2. Run MinIO with its original volume on a separate port, and start Garage with
   `docker compose up -d --wait s3`.
3. Configure [rclone S3 remotes](https://rclone.org/s3/) named `old-minio` and `garage`,
   then run `rclone copy old-minio:traq garage:traq --metadata` and
   `rclone check old-minio:traq garage:traq --download`.
4. After verification succeeds, run `mise run up` and check existing attachments.
   Keep the old volume until migration is confirmed; do not use `down -v`.
