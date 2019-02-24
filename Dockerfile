FROM golang:1.11.5-alpine AS build
ENV GO111MODULE=on
RUN apk add --update --no-cache git
WORKDIR /go/src/github.com/traPtitech/traQ
COPY ./go.* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /traQ


FROM alpine:3.9
WORKDIR /app

RUN apk add --update ca-certificates imagemagick openssl && \
    update-ca-certificates && \
    rm -rf /var/cache/apk/*
ENV DOCKERIZE_VERSION v0.6.1
RUN wget https://github.com/jwilder/dockerize/releases/download/$DOCKERIZE_VERSION/dockerize-alpine-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && tar -C /usr/local/bin -xzvf dockerize-alpine-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && rm dockerize-alpine-linux-amd64-$DOCKERIZE_VERSION.tar.gz

VOLUME /localstorage
EXPOSE 80
ENV TRAQ_PORT=80 \
    TRAQ_ORIGIN=http://localhost \
    TRAQ_IMAGEMAGICK_PATH=/usr/bin/convert \
    TRAQ_STORAGE_LOCAL_DIR=/localstorage

COPY ./static ./static/
COPY --from=build /traQ ./

ENTRYPOINT ./traQ
