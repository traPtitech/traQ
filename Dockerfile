FROM golang:1.15.7-alpine AS build
RUN apk add --update --no-cache git
WORKDIR /go/src/github.com/traPtitech/traQ
COPY ./go.* ./
RUN go mod download
COPY . .

ARG TRAQ_VERSION=dev
ARG TRAQ_REVISION=local
RUN CGO_ENABLED=0 go build -o /traQ -ldflags "-s -w -X main.version=$TRAQ_VERSION -X main.revision=$TRAQ_REVISION"

FROM alpine:3.13.0
WORKDIR /app

RUN apk add --update ca-certificates imagemagick && \
    update-ca-certificates && \
    rm -rf /var/cache/apk/*
ENV DOCKERIZE_VERSION v0.6.1
RUN wget https://github.com/jwilder/dockerize/releases/download/$DOCKERIZE_VERSION/dockerize-alpine-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && tar -C /usr/local/bin -xzvf dockerize-alpine-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && rm dockerize-alpine-linux-amd64-$DOCKERIZE_VERSION.tar.gz

VOLUME /app/storage
EXPOSE 3000
ENV TRAQ_IMAGEMAGICK=/usr/bin/convert

COPY --from=build /traQ ./

ENTRYPOINT ./traQ serve
