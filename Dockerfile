FROM golang:1.18.3-alpine AS build
WORKDIR /go/src/github.com/traPtitech/traQ
COPY ./go.* ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .

ENV GOCACHE=/tmp/go/cache
ARG TRAQ_VERSION=dev
ARG TRAQ_REVISION=local
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/tmp/go/cache CGO_ENABLED=0 go build -o /traQ -ldflags "-s -w -X main.version=$TRAQ_VERSION -X main.revision=$TRAQ_REVISION"

FROM golang:1.18.3-alpine AS dockerize

ARG DOCKERIZE_VERSION=v0.6.1
RUN go install github.com/jwilder/dockerize@$DOCKERIZE_VERSION

FROM alpine:3.16.0
WORKDIR /app

RUN apk add --no-cache --update ca-certificates imagemagick && \
    update-ca-certificates

VOLUME /app/storage
EXPOSE 3000
ENV TRAQ_IMAGEMAGICK=/usr/bin/convert

COPY --from=dockerize /go/bin/dockerize /usr/local/bin/
COPY --from=build /traQ ./

HEALTHCHECK CMD ./traQ healthcheck || exit 1
ENTRYPOINT ["./traQ"]
CMD ["serve"]
