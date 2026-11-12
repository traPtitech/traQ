# Development

If you want to contribute to traQ, then follow this section.

## Requirements

- git
- [mise](https://mise.jdx.dev/)

### Windows

Install PowerShell 7.2 or later and
[WSL Containers](https://learn.microsoft.com/en-us/windows/wsl/tutorials/wsl-containers).
Run the Windows commands in PowerShell. To run tests with the Go race
detector, install a C compiler such as MinGW-w64 and make sure `gcc` is on PATH.

### Linux and macOS

Install Bash and Docker with the Compose plugin (`docker compose`). To run
tests with the Go race detector, install a C compiler.

## Set up with mise

Use mise to install Go, wire, mockgen, and golangci-lint at the versions
specified in [`mise.toml`](../mise.toml).

```sh
mise trust
mise install
mise run init
```

The `init` task downloads Go module dependencies. Run `mise run help` to list
the available tasks.

## Run the local server

### Windows

#### WSL Containers

Run `mise run up` to start MariaDB with `wslc` and run the API in the
foreground at `http://localhost:3000`. Use `traq` for the initial username and
password.

Stop the API with Ctrl+C and run `mise run down` to stop MariaDB. Subsequent
server runs reuse the development database.

#### Docker Compose

For the bundled UI and search, install Docker with the Compose plugin. Run
`docker compose up -d --build` to start or rebuild the stack, and
`docker compose down` to stop it.

### Linux and macOS

Run `mise run up` to build and start the Docker Compose stack with the bundled
UI and search. Run the same command to rebuild traQ. Stop the stack with
`mise run down`.

### Docker Compose services

- traQ at `http://localhost:3000`. The initial username and password are both
  `traq`.
- Adminer at `http://localhost:3001`.
- traQ pprof at `http://localhost:6060`.
- MariaDB on port `3002/tcp` with username `root`, password `password`, and
  database `traq`.

## Remove development data

### Windows

#### WSL Containers

1. Stop the API with Ctrl+C and run `mise run down` to stop MariaDB.
2. Run `wslc volume rm traq-wslc-dev-db` to remove the development database
   volume.
3. Remove the `dev/data/windows/storage` directory to delete stored files.
4. Run `mise run up` to start with fresh data.

#### Docker Compose

1. Run `docker compose down --volumes` to remove the Compose volumes.
2. Run `docker compose up -d --build` to start with fresh data.

### Linux and macOS

1. Run `mise run down -- --volumes` to remove the Compose volumes.
2. Run `mise run up` to start with fresh data.

## Run tests

### Windows

Run `mise run test` to run the Go test suite with the race detector. It creates
a fresh MariaDB and RustFS for each run and removes both containers afterward.

### Linux and macOS

1. Run `mise run up-test-db` to start the test database.
2. Run `mise run test`.
3. Run `mise run rm-test-db` when you are done with the test database.

To run just the RustFS storage tests, use `mise exec -- go test ./utils/storage`
with Docker running.

## Build the executable

### Windows

Run `mise exec -- go build -o traQ.exe .`.

### Linux and macOS

Run `mise run traQ`.

## Regenerate Go code

Run `mise run gogen` to regenerate Wire and gomock files.

## Run linters

The Go linter uses [golangci-lint](https://github.com/golangci/golangci-lint).
The Swagger linter runs [Spectral](https://github.com/stoplightio/spectral)
through Docker.

### Windows

Run `mise run golangci-lint` in PowerShell. For Swagger lint, run
`mise run swagger-lint` in a WSL distribution with mise and Docker configured.

### Linux and macOS

Run `mise run lint` for both linters. To run them individually, use
`mise run golangci-lint` or `mise run swagger-lint`.

## Generate and lint database schema docs

If your changelist alters the database schema, update the descriptions in
`.tbls.yml` and regenerate the docs with [tbls](https://github.com/k1LoW/tbls).
The mise tasks run tbls through Docker with host networking. If you use Docker
Desktop, use version 4.34 or later and turn on **Enable host networking** under
**Settings > Resources > Network**.

### Windows

Run the schema tasks in a WSL distribution with mise and Docker configured.

### Linux and macOS

Run the schema tasks from your development shell with Docker running.

### Regenerate and lint the schema

1. Run `mise run up-test-db` to start the test database.
2. Run `mise run db-gen-docs` to regenerate the docs.
3. Run `mise run db-lint` to lint the schema.

## S3 storage with Docker Compose

The Docker Compose stack starts RustFS and creates the `traq` bucket with
the development credentials in `compose.yaml`. The S3 endpoint is
`http://localhost:9000` (`http://s3:9000` inside Compose), the web console is
`http://localhost:9001`, and the region is `ap-northeast-1`. Persistent data
is stored in the `rustfs` volume.

### Existing MinIO data

The old MinIO `s3` volume cannot be reused by RustFS. Complete the copy before
replacing the old MinIO deployment with this revision:

1. In the old revision, stop only the backend (`docker compose stop backend`) and
   back up the database and MinIO `s3` volume. Keep MinIO running as the source.
2. Provision RustFS at a separate endpoint and create its `traq` bucket.
3. Configure [rclone S3 remotes](https://rclone.org/s3/) named `old-minio` and
   `rustfs` with their explicit endpoints and region `ap-northeast-1`. Use S3
   provider `Minio` for `old-minio` and `Other` for `rustfs`, and set
   `force_path_style = true` for the RustFS remote.
4. Run `rclone copy old-minio:traq rustfs:traq --metadata`, followed by
   `rclone check old-minio:traq rustfs:traq --download` to verify object contents.
5. Export metadata with `rclone lsjson old-minio:traq --recursive --metadata` and
   `rclone lsjson rustfs:traq --recursive --metadata`, then compare each object's
   `Path`, `MimeType`, and the `content-type` and `content-disposition` entries in
   `Metadata`.
6. After verification succeeds, deploy this revision, point traQ at RustFS, start
   the backend, and check existing attachments. Keep the MinIO backup until the
   migration is confirmed.
