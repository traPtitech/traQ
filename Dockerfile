FROM golang:1.16-alpine AS build
WORKDIR /go/src/github.com/traPtitech/traQ
COPY ./go.* ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .

ENV GOCACHE=/tmp/go/cache
ARG TRAQ_VERSION=dev
ARG TRAQ_REVISION=local
RUN --mount=type=cache,target=/tmp/go/cache CGO_ENABLED=0 go build -o /traQ -ldflags "-s -w -X main.version=$TRAQ_VERSION -X main.revision=$TRAQ_REVISION"

FROM alpine:3.13.5
WORKDIR /app

RUN apk add --no-cache --update ca-certificates imagemagick && \
    update-ca-certificates
ENV DOCKERIZE_VERSION v0.6.1
RUN wget https://github.com/jwilder/dockerize/releases/download/$DOCKERIZE_VERSION/dockerize-alpine-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && tar -C /usr/local/bin -xzvf dockerize-alpine-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && rm dockerize-alpine-linux-amd64-$DOCKERIZE_VERSION.tar.gz

VOLUME /app/storage
EXPOSE 3000
ENV TRAQ_IMAGEMAGICK=/usr/bin/convert

COPY --from=build /traQ ./

HEALTHCHECK CMD ./traQ healthcheck || exit 1
ENTRYPOINT ["./traQ"]
CMD ["serve"]
