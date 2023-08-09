FROM --platform=$BUILDPLATFORM golang:1.21.0-alpine AS build
WORKDIR /go/src/github.com/traPtitech/traQ

COPY ./go.* ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

ENV GOCACHE=/tmp/go/cache
ENV CGO_ENABLED=0
ARG TRAQ_VERSION=dev
ARG TRAQ_REVISION=local

ARG TARGETOS
ARG TARGETARCH
ENV GOOS=$TARGETOS
ENV GOARCH=$TARGETARCH

RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/tmp/go/cache \
  go build -o /traQ -ldflags "-s -w -X main.version=$TRAQ_VERSION -X main.revision=$TRAQ_REVISION"

FROM alpine:3.18.3
WORKDIR /app

RUN apk add --no-cache --update ca-certificates imagemagick && \
  update-ca-certificates
ENV TRAQ_IMAGEMAGICK=/usr/bin/convert

COPY --from=build /traQ ./

VOLUME /app/storage
EXPOSE 3000

HEALTHCHECK CMD ./traQ healthcheck || exit 1
ENTRYPOINT ["./traQ"]
CMD ["serve"]
