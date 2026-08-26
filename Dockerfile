FROM --platform=$BUILDPLATFORM golang:1.27.0@sha256:0ecdc2a9f6156af6451080bfe3d8382a662fcc4e209608c6f919e643453514c1 AS build

RUN mkdir /storage

WORKDIR /go/src/github.com/traPtitech/traQ

COPY ./go.* ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

ENV GOCACHE=/tmp/go/cache
ENV CGO_ENABLED=0
ARG TRAQ_VERSION=dev
ARG TRAQ_REVISION=local

ARG TARGETOS
ARG TARGETARCH
ENV GOOS=$TARGETOS
ENV GOARCH=$TARGETARCH

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/tmp/go/cache \
  go build -o /traQ -ldflags "-s -w -X main.version=$TRAQ_VERSION -X main.revision=$TRAQ_REVISION"

FROM gcr.io/distroless/base-debian12@sha256:fabbf1c0c357a3d42550111351daed089b20a2c954df13ee2fcff60602515e84
WORKDIR /app
EXPOSE 3000

COPY --from=build /storage/ /app/storage/
VOLUME /app/storage

COPY --from=build /traQ ./

HEALTHCHECK CMD ["./traQ", "healthcheck", "||", "exit", "1"]
ENTRYPOINT ["./traQ"]
CMD ["serve"]
