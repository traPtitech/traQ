FROM --platform=$BUILDPLATFORM golang:1.27.2@sha256:5bc7f572bbaa98885a3a1fd9c0aa76b59e3e14e8628bfc316bbfd0c701e4818c AS build

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

FROM debian:bookworm-slim@sha256:7ff6ba31527b9e336162fa061746c366c4b2032f4d2cff2078bea43f78efe3f8
RUN apt-get update && \
  apt-get install -y --no-install-recommends ca-certificates ffmpeg && \
  rm -rf /var/lib/apt/lists/*
WORKDIR /app
EXPOSE 3000

COPY --from=build /storage/ /app/storage/
VOLUME /app/storage

COPY --from=build /traQ ./

HEALTHCHECK CMD ["./traQ", "healthcheck", "||", "exit", "1"]
ENTRYPOINT ["./traQ"]
CMD ["serve"]
