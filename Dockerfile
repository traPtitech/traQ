FROM golang:1.19.5 AS build
SHELL ["/bin/bash", "-o", "pipefail", "-c"]

RUN mkdir /storage

ARG DOCKERIZE_VERSION=v0.6.1
RUN go install github.com/jwilder/dockerize@$DOCKERIZE_VERSION

WORKDIR /go/src/github.com/traPtitech/traQ
COPY ./go.* ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .

ENV CGO_ENABLED=0
ENV GOCACHE=/tmp/go/cache
ARG TRAQ_VERSION=dev
ARG TRAQ_REVISION=local
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/tmp/go/cache \
  go build -o /traQ -ldflags "-s -w -X main.version=$TRAQ_VERSION -X main.revision=$TRAQ_REVISION"

FROM debian:bullseye-20230208-slim AS convert

RUN apt-get update && apt-get install --no-install-recommends -y imagemagick

FROM gcr.io/distroless/base:nonroot
WORKDIR /app
EXPOSE 3000

# ボリュームにマウントするディレクトリは、nonrootユーザーが所有するように
COPY --chown=nonroot:nonroot --from=build /storage/ /app/storage/
VOLUME /app/storage

COPY --chown=nonroot:nonroot --from=build /go/bin/dockerize /usr/local/bin/
COPY --chown=nonroot:nonroot --from=build /traQ ./
COPY --from=convert /usr/lib/ /usr/lib/
COPY --from=convert /lib/ /lib/
COPY --from=convert /usr/bin/convert /usr/bin/
ENV TRAQ_IMAGEMAGICK=/usr/bin/convert

HEALTHCHECK CMD ./traQ healthcheck || exit 1
ENTRYPOINT ["./traQ"]
CMD ["serve"]
