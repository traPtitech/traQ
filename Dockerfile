FROM --platform=$BUILDPLATFORM golang:1.26.3@sha256:2d6c80227255c3112a4d08e67ba98e58efd3846daf15d9d7d4c389565d881b1a AS build

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

FROM gcr.io/distroless/base-debian12@sha256:58695f439f772a00009c8f6be4c183f824c1f556d74b313c30900f167e4772f8
WORKDIR /app
EXPOSE 3000

COPY --from=build /storage/ /app/storage/
VOLUME /app/storage

COPY --from=build /traQ ./

HEALTHCHECK CMD ["./traQ", "healthcheck", "||", "exit", "1"]
ENTRYPOINT ["./traQ"]
CMD ["serve"]
